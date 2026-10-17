#!/usr/bin/env python3
"""Every Go module fetch in a sample's Dockerfile runs in a bounded retry loop.

proxy.golang.org now and then resets a download mid-transfer ("stream error:
stream ID N; INTERNAL_ERROR"), and the next attempt gets it. keploy's CI
clones this repo at main and builds these Dockerfiles, so one fetch made only
once fails a keploy lane that has nothing to do with the sample (keploy
enterprise pipeline 11675: go-grpc's `RUN go mod tidy`).

For each Dockerfile, in each stage:
  - a fetch (`go mod download`, `go mod tidy`, `go mod vendor`, any `go get`,
    or `go install`/`go run` of a module at a version) is the condition of an
    `until` (or `while !`) loop, in the RUN's own shell (not in a subshell or
    an sh -c, whose exit leaves only that shell), that counts its tries and
    fails the build when they run out, after 2 to 10: `RUN n=0; until go mod
    download; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; ...; done`. A fetch
    with -e is refused: it succeeds on a fetch that failed.
  - the stage's first build (go build, test, run, vet, generate, list, tool,
    or install of the module itself) comes after such a loop of `go mod
    download`/`tidy`/`vendor` for its module: in its WORKDIR or one above,
    in the stage or one it is built FROM, or in a stage whose whole module
    cache it COPYs to its own. A build fetches what the module cache lacks,
    once; a tool installed at a version fetches nothing of the module's.
  - a loop in a RUN with a cache mount at the module cache fills that mount,
    which only a RUN with the same mount sees; such a mount hides the image's
    own module cache.
go is found alone or by its path (/usr/local/go/bin/go), in backticks, and in
the script of an `sh -c`/`bash -c`. A loop counts only in the RUN's own shell:
not in a subshell, a pipeline or the background. A RUN this cannot read (exec
form, a here-document, go in a quoted string run by anything but echo or
printf: su -c, eval, or piped into a shell) is refused, not passed: it cannot
be verified. Not read at all: what a script or a Makefile the RUN runs does.

Run with no arguments to check the repo; `--self-test` checks the rule on
fixtures first. Exits 1 and names each line on a failure.
"""
import os
import posixpath
import re
import subprocess
import sys

# Where a command starts, and `go` as one: alone, or by its path
# (/usr/local/go/bin/go).
AT = r'(?:^|(?<=[\s;&|(){}`]))'
GO = r'(?:\S*/bin/)?go\s+'
# -e makes go mod download/tidy/vendor succeed on a fetch that failed: a loop
# on one never retries, and the build then fails on what it left out.
MODULE_FETCH = r'go\s+mod\s+(?:download|tidy|vendor)(?:\s+-(?!e\b)[\w=.,/-]*)*'
# Fetches by name: go install or run of a module at a version, and any go get
# (which adds and upgrades requirements, -u fetching new versions).
BY_NAME = r'go\s+(?:install|run)\s+(?:-\S+\s+)*[^\s;&|]*@[^\s;&|]+|go\s+get\b[^;&|]*'
ANY_FETCH = re.compile(AT + r'((?:\S*/bin/)?(?:' + MODULE_FETCH + '|' + BY_NAME + r'))')
E_FLAG = re.compile(AT + GO + r'mod\s+(?:download|tidy|vendor)\b[^;&|]*?\s-e\b')
# A build: it fetches what the module cache lacks, once.
BUILD = re.compile(AT + GO + r'(build|test|run|vet|generate|list|install|tool|mod\s+(?:why|graph))\b')
AT_VERSION = re.compile(r'(?:install|run)\s+(?:-\S+\s+)*[^\s;&|]*@')
# A fetch as the condition of an until (or `while !`) loop, under env
# assignments and a `timeout N` if any; its body runs to the first `; done`.
LOOPED = (AT + r'(?:until|while\s+!)\s+(?:\w+=\S*\s+)*(?:timeout\s+(?:-\S+\s+)*\d+[smhd]?\s+)?(?:\S*/bin/)?(?:FETCH)'
          r'\s*;\s*do\b(?P<body>.*?);\s*done\b')
LOOPED_MODULE = re.compile(LOOPED.replace('FETCH', MODULE_FETCH))
LOOPED_ANY = re.compile(LOOPED.replace('FETCH', MODULE_FETCH + '|' + BY_NAME))
# The body's bound: a test of a counter that fails the build (exit K, K > 0)
# once it reaches N: `[ "$n" -lt N ] || exit 1`, `[ "$n" -ge N ] && exit 1`,
# `if [ "$n" -ge N ]; then exit 1`, with `test` for `[`, and `|| { ...; exit 1; }`.
BOUND = re.compile(r'(?:\[\[?\s*|\btest\s+)"?\$\{?(?P<v>\w+)\}?"?\s+-(?P<op>lt|le|ge|gt|eq)\s+(?P<n>\d+)\s*(?:\]\]?)?\s*'
                   r'(?:(?P<join>\|\||&&)\s*(?:\{\s*(?:[^;{}()]*;\s*)*?)?|;\s*then\s+(?:(?!\s*(?:fi|else|elif)\b)[^;{}()]*;\s*)*?)exit\s+[1-9]\d*\b')
# How many tries each bound allows, with the counter counted up before it.
TRIES = {'lt': 0, 'ge': 0, 'eq': 0, 'le': 1, 'gt': 1}
MAX_TRIES = 10
CACHE_MOUNT = re.compile(r'--mount=(?=\S*\btype=cache\b)(\S+)')
# sh -c 'script': bash, dash or sh, any flags (-e, -o pipefail, --login), -c,
# an optional --.
SH_C = re.compile(r'\b(?:ba|da)?sh\s+(?:(?:[-+]o\s+\w+|--?[\w-]+)\s+)*?-\w*c\w*\s+(?:--\s+)?("(?:[^"\\]|\\.)*"|\'[^\']*\')')
ARITH = re.compile(r'\$\(\((?:[^()]|\([^()]*\))*\)\)')
QUOTED = re.compile(r'"(?:[^"\\]|\\.)*"|\'[^\']*\'')
VAR_QUOTED = re.compile(r'"\$\{?\w+\}?"')
MOD_CACHE = re.compile(r'(?:\S*/pkg/mod|\$\{?GOMODCACHE\}?)/?')


class Body:
    """A RUN's shell text, read for code. Each view keeps every offset of the
    raw text, so a span in one is the same text in another.
      raw:    as written;
      code:   the script of an sh -c read as code, $((...)) and quoted text
              blanked, a trailing comment cut;
      bound:  raw with $((...)) kept, and quoted text blanked but for a quoted
              variable ("$n"), an sh -c script with it: a loop in one is not
              bounded, as its exit leaves only that shell;
      quoted: (span, inner text) of each quoted string left in code."""

    def __init__(self, raw):
        self.raw = raw
        s = SH_C.sub(lambda m: m.group(0)[:m.start(1) - m.start(0)] + ' ' + m.group(1)[1:-1] + ' ', raw)
        s = ARITH.sub(lambda m: ' ' * len(m.group(0)), s)
        self.quoted = [(m.span(), m.group(0)[1:-1]) for m in QUOTED.finditer(s)]
        s = QUOTED.sub(lambda m: '"' + 'x' * (len(m.group(0)) - 2) + '"', s)
        self.code = re.sub(r'(?:^|\s)#.*$', '', s)
        self.bound = QUOTED.sub(lambda m: m.group(0) if VAR_QUOTED.fullmatch(m.group(0))
                                else '"' + 'x' * (len(m.group(0)) - 2) + '"', raw)

    def depth(self, pos):
        """how many ( subshells and $( substitutions enclose pos: an exit there
        leaves only that shell, not the RUN."""
        c = self.code[:pos]
        return c.count('(') - c.count(')')

    def command(self, pos):
        """the first word of the simple command around pos (past assignments)."""
        piece = re.split(r'[;|(){}`]|(?<![<>])&(?!>)|\b(?:then|do|else)\b', self.code[:pos])[-1]
        words = [w for w in piece.split() if not re.fullmatch(r'\w+=\S*|\d*[<>]\S*', w)]
        return words[0] if words else ''


def bounded(b, body_span):
    """the loop body at body_span (in b's views) gives up with a failure after
    2 to MAX_TRIES counted tries, from the RUN's own shell."""
    a, z = body_span
    view = b.bound[a:z]
    for m in BOUND.finditer(view):
        v, op, n = m.group('v'), m.group('op'), int(m.group('n'))
        if (op in ('lt', 'le')) != (m.group('join') == '||'):  # (no join: `then`)
            continue  # a bound the wrong way round: it gives up at the first try, or never
        if not 2 <= n + TRIES[op] <= MAX_TRIES:
            continue
        if b.depth(a + m.start()) != b.depth(a):
            continue  # its exit leaves a subshell, not the loop's shell
        if not any(b.depth(a + i.start()) == b.depth(a) for i in
                   re.finditer(r'(?<![\w$])' + v + r'=\$\(\(\s*\$?' + v + r'\s*\+\s*1\s*\)\)', view)):
            continue  # nothing counts the tries in the loop's shell
        if re.search(r'(?<![\w$])' + v + r'=\d', view):
            continue  # the count starts again each time round
        return True
    return False


def past_redirections(text):
    """text past the redirections that start it (2>&1, >/dev/null, > /tmp/log)."""
    return re.sub(r'^(?:\s*\d*[<>]+&?(?:\s*[^\s;&|<>]+)?)*', '', text).lstrip()


def brace_groups(c):
    """(start, end) in c of each `{ ...; }` group."""
    out, opens = [], []
    for m in re.finditer(r'(?:^|(?<=[\s;&|(]))\{(?=\s)|(?<=[;\s])\}(?=\s|$|[;&|)<>])', c):
        if m.group(0) == '{':
            opens.append(m.start())
        elif opens:
            out.append((opens.pop(), m.end()))
    return out


def own_shell(c, a, z):
    """the command at c[a:z] runs in the RUN's own shell, so its exit ends the
    RUN: neither it nor a brace group around it is in a pipeline (whose
    commands each run in a subshell) or in the background."""
    for s, e in [(a, z)] + [g for g in brace_groups(c) if g[0] < a and g[1] >= z]:
        before, after = c[:s].rstrip(), past_redirections(c[e:])
        if before.endswith('|') and not before.endswith('||'):
            return False
        if after.startswith('|') and not after.startswith('||'):
            return False
        if after.startswith('&') and not after.startswith('&&'):
            return False
    return True


def looped(regex, b):
    """spans in b.code of each fetch loop regex finds that is bounded and runs
    in the RUN's own shell, so its exit fails the RUN: not in a subshell, a
    pipeline, or the background."""
    out = []
    for m in regex.finditer(b.code):
        if b.depth(m.start()) or not own_shell(b.code, m.start(), m.end()):
            continue
        if bounded(b, m.span('body')):
            out.append(m.span())
    return out


def mod_cache_mounts(body):
    """the BuildKit cache mounts (by id, or target) a RUN puts at the module cache."""
    out = set()
    for m in CACHE_MOUNT.finditer(body):
        opts = dict(o.split('=', 1) for o in m.group(1).split(',') if '=' in o)
        target = opts.get('target', opts.get('dst', opts.get('destination', '')))
        if MOD_CACHE.fullmatch(target):
            out.add(opts.get('id', target))
    return out


def covered(dirs, wd):
    """a fetch made in one of dirs (its go.mod) serves a build in wd."""
    return any(wd == d or wd.startswith(d.rstrip('/') + '/') for d in dirs)


def instructions(text):
    """(first line, last line, instruction) of a Dockerfile, continuations joined."""
    out, buf, start = [], '', 0
    # (rstrip takes a CRLF line's \r with its other trailing space)
    for n, line in enumerate(text.split('\n'), 1):
        if not buf:
            start = n
            if line.lstrip().startswith('#'):
                continue
        elif line.lstrip().startswith('#') or not line.strip():
            continue  # Docker drops a comment or blank line inside a continuation
        if line.rstrip().endswith('\\'):
            buf += line.rstrip()[:-1] + ' '
            continue
        out.append((start, n, buf + line))
        buf = ''
    if buf:
        out.append((start, n, buf))
    return out


def problems(name, text):
    out = []
    # per stage: the directories whose module a looped download fetched into
    # the image's layers, and the WORKDIR
    stages, cur, fetched, wd, reported = {}, '', set(), '/', False
    # per cache mount at the module cache: the directories a looped download
    # in a RUN with that mount fetched for; only a RUN with the mount sees it
    mounts_fetched = {}
    lines = text.replace('\r\n', '\n').split('\n')
    # the last line of a here-document's body and delimiter: the lines up to it
    # are the document's, however a continuation in it joined them
    skip_to = 0
    for n, last, ins in instructions(text):
        if n <= skip_to:
            continue
        words = ins.split()
        if not words:
            continue
        kind = words[0].upper()
        if kind == 'FROM':
            args = [a.lower() for a in words[1:] if not a.startswith('--')]
            if cur:
                stages[cur] = (fetched, wd)
            fetched, wd = stages.get(args[0], (set(), '/')) if args else (set(), '/')
            fetched, reported = set(fetched), False
            cur = args[2] if len(args) == 3 and args[1] == 'as' else ''
            continue
        moved = [m for m in re.finditer(r'\b(GOMODCACHE|GOPATH)=("?)([^\s"]*)\2', ins)
                 if posixpath.normpath(m.group(3) or '.') != {'GOPATH': '/go', 'GOMODCACHE': '/go/pkg/mod'}[m.group(1)]]
        if kind in ('ENV', 'ARG') and moved:
            fetched = set()  # a module cache elsewhere: what was fetched is not in it
            continue
        if kind == 'WORKDIR' and len(words) > 1:
            wd = posixpath.normpath(posixpath.join(wd, words[1]))
            continue
        if kind in ('COPY', 'ADD'):
            # a here-document's body is the file's, not instructions
            for m in re.finditer(r'(?<!<)<<-?\s*(["\']?)([A-Za-z_]\w*)\1', ins):
                for i in range(last, len(lines)):
                    skip_to = i + 1
                    if lines[i].strip() == m.group(2):
                        break
                break
        if kind == 'COPY':
            src = [w for w in words[1:] if w.startswith('--from=')]
            paths = [w for w in words[1:] if not w.startswith('--')]
            # the whole module cache, to the same place
            if src and len(paths) == 2 and MOD_CACHE.fullmatch(paths[1]) and \
                    posixpath.normpath(paths[0]) == posixpath.normpath(paths[1]):
                fetched |= stages.get(src[0].split('=', 1)[1].lower(), (set(), '/'))[0]
            continue
        if kind != 'RUN':
            continue
        raw = ins.split(None, 1)[1] if len(words) > 1 else ''
        b = Body(raw)
        # a here-document's << outside quotes and $((...)), its delimiter as written
        h = next((m for m in re.finditer(r'(?<!<)<<-?\s*(["\']?)([A-Za-z_]\w*)\1', raw)
                  if b.code[m.start():m.start() + 2] == '<<'), None)
        if re.match(r'\s*\[\s*"', raw) or h:
            if h:
                hd = []
                for i in range(last, len(lines)):
                    skip_to = i + 1
                    if lines[i].strip() == h.group(2):
                        break
                    hd.append(lines[i])
                raw += '\n' + '\n'.join(hd)
            if re.search(r'\bgo\b', raw):
                out.append(f'{name}:{n}: a RUN in exec form or with a here-document runs go, and this '
                           'cannot verify it: write it as a shell-form RUN with the retry loop')
            continue
        c = b.code
        to_shell = re.search(r'(?<!\|)\|\s*(?:(?:\S*/)?xargs\b[^|;&]*?\s)?(?:\S*/)?(?:ba|da)?sh\b', c)
        for (qa, _), inner in b.quoted:
            if not (BUILD.search(inner) or ANY_FETCH.search(inner)):
                continue
            who = b.command(qa)
            if who in ('echo', 'printf') and not to_shell:
                continue  # a message
            who = 'the shell it is piped into' if who in ('echo', 'printf') else f'`{who or "a command"}`'
            out.append(f'{name}:{n}: {who} runs go from a quoted string, and this cannot verify it: run go in '
                       'the RUN itself, with the retry loop')
        looped_any = looped(LOOPED_ANY, b)
        mounts = mod_cache_mounts(raw)
        if looped(LOOPED_MODULE, b):
            if mounts:
                for k in mounts:
                    mounts_fetched.setdefault(k, set()).add(wd)
            else:
                fetched.add(wd)
        # a cache mount at the module cache hides the layers' module cache
        seen = covered(set().union(*(mounts_fetched.get(k, set()) for k in mounts)), wd) if mounts else covered(fetched, wd)
        e_flags = [m.start() for m in E_FLAG.finditer(c)]
        for m in E_FLAG.finditer(c):
            out.append(f'{name}:{n}: `{m.group(0).strip()}` succeeds on a fetch that failed, so its loop never '
                       'retries: drop -e')
        for m in ANY_FETCH.finditer(c):
            if m.start() in e_flags:
                continue
            if not any(a <= m.start(1) < z for a, z in looped_any):
                out.append(f'{name}:{n}: `{m.group(1).strip()}` is not the condition of an until loop, in the RUN\'s '
                           'own shell, that counts its tries and fails the build when they run out')
        for m in BUILD.finditer(c):
            if AT_VERSION.match(c[m.start(1):]):
                continue  # a fetch by name, checked above
            if not seen and not reported:
                out.append(f'{name}:{n}: `go {m.group(1)}` fetches what the module cache lacks, once: run `go mod '
                           f'download` in a retry loop first, for this module ({wd}), in its stage or one it is '
                           'built FROM')
                reported = True
    return out


LOOP = 'until go mod download; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; sleep $((n*5)); done'
SELF_TEST = {
    'bare build': ('FROM golang:1.23\nRUN go build -o /x .\n', 1),
    'each build verb': ('FROM golang:1.23\nRUN go test ./...\nFROM golang:1.23\nRUN go vet .\nFROM golang:1.23\nRUN go run .\n'
                        'FROM golang:1.23\nRUN go generate\nFROM golang:1.23\nRUN go list ./...\n', 5),
    'one report per stage': ('FROM golang:1.23\nRUN go build -o /a .\nRUN go build -o /b .\n', 1),
    'bare download': ('FROM golang:1.23\nRUN go mod download\nRUN go build .\n', 2),
    'looped download': ('FROM golang:1.23\nRUN n=0; ' + LOOP + '\nRUN go build .\n', 0),
    'looped tidy, its loop over three lines': ('FROM golang:1.23\nRUN n=0; \\\n  until go mod tidy; \\\n  do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; done\nRUN go build .\n', 0),
    'a CRLF continuation': ('FROM golang:1.23\r\nRUN n=0; \\\r\n  ' + LOOP + '\r\nRUN go build .\r\n', 0),
    'a loop in a comment': ('FROM golang:1.23\nRUN go mod download # until go mod download; do n=1; [ x ] || exit 1; done\nRUN go build .\n', 2),
    'loop words in an echo': ('FROM golang:1.23\nRUN echo "until done || exit 1" && go mod download\n', 1),
    'a loop, then a bare fetch': ('FROM golang:1.23\nRUN for f in a b; do echo $f; done; go mod download\n', 1),
    'a loop that never gives up': ('FROM golang:1.23\nRUN while true; do go mod download && break; done\n', 1),
    'an until loop that never gives up': ('FROM golang:1.23\nRUN until go mod download; do sleep 5; done\n', 1),
    'a loop whose exit is not counted': ('FROM golang:1.23\nRUN until go mod download; do sleep 5 || exit 1; done\n', 1),
    'a loop that gives up at once': ('FROM golang:1.23\nRUN until go mod download; do exit 1; done\n', 1),
    'a bound on another counter': ('FROM golang:1.23\nRUN until go mod download; do m=$((m+1)); [ "$n" -lt 5 ] || exit 1; done\n', 1),
    'a bound the wrong way round': ('FROM golang:1.23\nRUN until go mod download; do n=$((n+1)); [ "$n" -ge 5 ] || exit 1; done\n', 1),
    'a bound that gives up at the first try': ('FROM golang:1.23\nRUN until go mod download; do n=$((n+1)); [ "$n" -lt 5 ] && exit 1; done\n', 1),
    'a bound by && exit': ('FROM golang:1.23\nRUN n=0; until go mod download; do n=$((n+1)); [ "$n" -ge 5 ] && exit 1; sleep 1; done\nRUN go build .\n', 0),
    'a cache mount on the download alone': ('FROM golang:1.23\nRUN --mount=type=cache,target=/go/pkg/mod n=0; ' + LOOP + '\nRUN go build .\n', 1),
    'a cache mount on the download and the build': ('FROM golang:1.23\nRUN --mount=type=cache,target=/go/pkg/mod n=0; ' + LOOP + '\n'
                                                    'RUN --mount=type=cache,target=/go/pkg/mod go build .\n', 0),
    'a cache mount on the build alone': ('FROM golang:1.23\nRUN n=0; ' + LOOP + '\nRUN --mount=type=cache,target=/go/pkg/mod go build .\n', 1),
    'a cache mount by id': ('FROM golang:1.23\nRUN --mount=type=cache,id=gomod,target=/go/pkg/mod n=0; ' + LOOP + '\n'
                            'FROM golang:1.23\nRUN --mount=type=cache,id=gomod,target=/root/go/pkg/mod go build .\n', 0),
    'a stage built FROM a fetching one': ('FROM golang:1.23 AS deps\nRUN n=0; ' + LOOP + '\nFROM deps\nRUN go build .\n', 0),
    'a fresh stage after a fetching one': ('FROM golang:1.23 AS deps\nRUN n=0; ' + LOOP + '\nFROM golang:1.23\nRUN go build .\n', 1),
    'FROM --platform': ('FROM --platform=$BUILDPLATFORM golang:1.23 AS deps\nRUN n=0; ' + LOOP + '\nFROM --platform=$BUILDPLATFORM deps\nRUN go build .\n', 0),
    'the module cache copied from a fetching stage': ('FROM golang:1.23 AS deps\nRUN n=0; ' + LOOP + '\nFROM golang:1.23\nCOPY --from=deps /go/pkg/mod /go/pkg/mod\nRUN go build .\n', 0),
    'a tool at a version, once': ('FROM golang:1.23\nRUN go install github.com/x/y@v1.0.0\n', 1),
    'a tool looped, then a build': ('FROM golang:1.23\nRUN n=0; until go install github.com/x/y@v1.0.0; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; done\nRUN go build .\n', 1),
    'go run at a version, once': ('FROM golang:1.23\nRUN go run github.com/x/gen@v1.0.0\n', 1),
    'go mod vendor, once': ('FROM golang:1.23\nRUN go mod vendor\n', 1),
    'a fetch after &&': ('FROM golang:1.23\nRUN cd /app && go mod download && go build .\n', 2),
    'go in a quoted message': ('FROM golang:1.23\nRUN echo "now go build it"\n', 0),
    'go in another instruction': ('FROM golang:1.23\nCMD ["go", "build"]\nLABEL x="go build"\n', 0),
    'exec form': ('FROM golang:1.23\nRUN ["go", "mod", "download"]\n', 1),
    'a here-document': ('FROM golang:1.23\nRUN <<EOT\ngo mod download\ngo build .\nEOT\n', 1),
    'no go': ('FROM alpine\nRUN echo hi\n', 0),
    'a build through sh -c': ('FROM golang:1.23\nRUN sh -c "go build -o /x ."\n', 1),
    'a fetch and build through bash -c': ('FROM golang:1.23\nRUN bash -c \'go mod download && go build .\'\n', 2),
    'go by its path': ('FROM debian\nRUN /usr/local/go/bin/go mod download\nRUN /usr/local/go/bin/go build -o /x .\n', 2),
    'go by its path, looped': ('FROM debian\nRUN n=0; until /usr/local/go/bin/go mod download; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; done\nRUN /usr/local/go/bin/go build .\n', 0),
    'go install of the module': ('FROM golang:1.23\nCOPY . .\nRUN go install ./cmd/server\n', 1),
    'go get of the module': ('FROM golang:1.23\nRUN go get -d -v ./...\n', 1),
    'go tool': ('FROM golang:1.23\nRUN go tool cover -h\n', 1),
    'a comment inside a continuation': ('FROM golang:1.23\nRUN apt-get update && \\\n    # fetch and build\n    go mod download && go build .\n', 2),
    'a blank line inside a continuation': ('FROM golang:1.23\nRUN apt-get update && \\\n\n    go build .\n', 1),
    'a loop that exits 0': ('FROM golang:1.23\nRUN until go mod download; do n=$((n+1)); [ "$n" -lt 5 ] || exit 0; done\n', 1),
    'a bound in a later loop': ('FROM golang:1.23\nRUN until go mod download; do sleep 5; done; n=0; while true; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; done\n', 1),
    'a module cache from a stage that fetched nothing': ('FROM golang:1.23 AS deps\nRUN echo none\nFROM golang:1.23\nCOPY --from=deps /go/pkg/mod /go/pkg/mod\nRUN go build .\n', 1),
    'a looped go run at a version': ('FROM golang:1.23\nRUN n=0; until go run github.com/x/gen@v1.0.0; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; done\n', 0),
    'a looped download with flags and env': ('FROM golang:1.23\nRUN n=0; until GOFLAGS=-mod=mod go mod download -x; do n=$((n+1)); if [ "$n" -ge 5 ]; then exit 1; fi; done\nRUN go build .\n', 0),
    'a while ! loop': ('FROM golang:1.23\nRUN n=0; while ! go mod download; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; done\nRUN go build .\n', 0),
    'a looped vendor': ('FROM golang:1.23\nRUN n=0; until go mod vendor; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; done\nRUN go build .\n', 0),
    'arithmetic is no here-document': ('FROM golang:1.23\nRUN s=2; echo $((1<<s))\nRUN go build .\nFROM golang:1.23\nRUN go build .\n', 2),
    'a here-string is no here-document': ('FROM golang:1.23\nRUN tr a b <<< "abc"\nRUN go build .\nFROM golang:1.23\nRUN go build .\n', 2),
    'bash -o pipefail -c': ('FROM golang:1.23\nRUN bash -o pipefail -c \'go mod download && go build .\'\n', 2),
    'sh --login -c --': ('FROM golang:1.23\nRUN sh --login -c -- "go build ."\n', 1),
    'su -c runs go from a quoted string': ('FROM golang:1.23\nRUN su app -c "go build -o /x ."\n', 1),
    'eval runs go from a quoted string': ('FROM golang:1.23\nRUN eval "go build -o /x ."\n', 1),
    'printf of go is no build': ('FROM golang:1.23\nRUN printf "%s\\n" "go mod download"\n', 0),
    'go mod tidy -e in a loop': ('FROM golang:1.23\nRUN n=0; until go mod tidy -e; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; done\nRUN go build .\n', 2),
    'go mod download -e=true, looped': ('FROM golang:1.23\nRUN n=0; until go mod download -e=true; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; done\n', 1),
    'the module cache copied elsewhere': ('FROM golang:1.23 AS deps\nRUN n=0; ' + LOOP + '\nFROM golang:1.23\nCOPY --from=deps /go/pkg/mod /tmp/unused\nRUN go build .\n', 1),
    'part of the module cache copied': ('FROM golang:1.23 AS deps\nRUN n=0; ' + LOOP + '\nFROM golang:1.23\n'
                                        'COPY --from=deps /go/pkg/mod/github.com/x /go/pkg/mod/github.com/x\nRUN go build .\n', 1),
    'a bound in a quoted echo': ('FROM golang:1.23\nRUN until go mod download; do echo "n=$((n+1)) [ $n -lt 5 ] || exit 1"; done\n', 1),
    'a bound in a subshell': ('FROM golang:1.23\nRUN until go mod download; do n=$((n+1)); ( [ "$n" -lt 5 ] || exit 1 ); done\n', 1),
    'a count that starts again': ('FROM golang:1.23\nRUN until go mod download; do n=0; n=$((n+1)); [ "$n" -lt 5 ] || exit 1; done\n', 1),
    'a bound past ten tries': ('FROM golang:1.23\nRUN until go mod download; do n=$((n+1)); [ "$n" -lt 1000000 ] || exit 1; done\n', 1),
    'a bound of one try': ('FROM golang:1.23\nRUN until go mod download; do n=$((n+1)); [ "$n" -lt 1 ] || exit 1; done\n', 1),
    'a loop in sh -c, its failure ignored': ('FROM golang:1.23\nRUN sh -c \'n=0; ' + LOOP + '\' || true; go build .\n', 2),
    'a loop in a subshell': ('FROM golang:1.23\nRUN (n=0; ' + LOOP + '); go build .\n', 2),
    'go get -u after a loop': ('FROM golang:1.23\nRUN n=0; ' + LOOP + '\nRUN go get -u ./...\n', 1),
    'go get looped at a version': ('FROM golang:1.23\nRUN n=0; until go get github.com/x/y@v1.2.3; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; done\n', 0),
    'a build in another module': ('FROM golang:1.23\nWORKDIR /a\nRUN n=0; ' + LOOP + '\nWORKDIR /b\nRUN go build .\n', 1),
    'a build below the fetched module': ('FROM golang:1.23\nWORKDIR /a\nRUN n=0; ' + LOOP + '\nWORKDIR cmd/server\nRUN go build .\n', 0),
    'a stage FROM a fetching one keeps its WORKDIR': ('FROM golang:1.23 AS deps\nWORKDIR /app\nRUN n=0; ' + LOOP + '\nFROM deps\nRUN go build .\n', 0),
    'a quoted here-document delimiter': ('FROM golang:1.23\nRUN cat > /cfg <<\'EOF\'\nkey=1\nEOF\nRUN n=0; ' + LOOP + '\nRUN go build .\n', 0),
    'done echoed in the loop': ('FROM golang:1.23\nRUN n=0; until go mod download; do n=$((n+1)); echo done; [ "$n" -lt 5 ] || exit 1; done\nRUN go build .\n', 0),
    'a bound by || { ...; exit 1; }': ('FROM golang:1.23\nRUN n=0; until go mod download; do n=$((n+1)); [ "$n" -lt 5 ] || { echo "gave up"; exit 1; }; sleep 1; done\nRUN go build .\n', 0),
    'a bound by test': ('FROM golang:1.23\nRUN n=0; until go mod download; do n=$((n+1)); test "$n" -lt 5 || exit 1; done\nRUN go build .\n', 0),
    'a go directory is no go': ('FROM golang:1.23\nRUN mkdir -p /go test\n', 0),
    'a loop after ||': ('FROM golang:1.23\nRUN n=0; [ -d /nope ] || ' + LOOP + '\nRUN go build .\n', 0),
    'a loop in a group fed by a pipe, after a group of its own': ('FROM golang:1.23\nRUN echo x | { { echo a; }; n=0; ' + LOOP + '; }\nRUN go build .\n', 2),
    'an exit after a then-block that ends': ('FROM golang:1.23\nRUN until go mod download; do n=$((n+1)); if [ "$n" -ge 5 ]; then echo x; fi; exit 1; done\n', 1),
    'an exit after an else': ('FROM golang:1.23\nRUN until go mod download; do n=$((n+1)); if [ "$n" -ge 5 ]; then :; else exit 1; fi; done\n', 1),
    'a here-document body after a continued delimiter': ('FROM golang:1.23\nRUN cat <<EOF > /x \\\nEOF\nRUN n=0; until go mod download; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; done\nEOF\nRUN go build .\n', 2),
    'a loop logged with spaces, in the background': ('FROM golang:1.23\nRUN n=0; ' + LOOP + ' > /tmp/dl.log 2>&1 &\nRUN go build .\n', 2),
    'a loop with its errors logged, piped': ('FROM golang:1.23\nRUN n=0; ' + LOOP + ' 2> /tmp/err | tee /tmp/dl.log\nRUN go build .\n', 2),
    'a piped brace group': ('FROM golang:1.23\nRUN { n=0; ' + LOOP + '; } 2>&1 | tee /tmp/dl.log\nRUN go build .\n', 2),
    'a brace group fed by a pipe': ('FROM golang:1.23\nRUN echo x | { n=0; ' + LOOP + '; }\nRUN go build .\n', 2),
    'a brace group alone': ('FROM golang:1.23\nRUN { n=0; ' + LOOP + '; }\nRUN go build .\n', 0),
    'a COPY here-document that says FROM': ('FROM golang:1.23\nRUN n=0; ' + LOOP + '\nCOPY <<EOF /notes\nFROM deps\nEOF\nRUN go build .\n', 0),
    'a COPY here-document that holds the loop': ('FROM golang:1.23\nCOPY <<EOF /notes\nRUN n=0; until go mod download; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; done\nEOF\nRUN go build .\n', 1),
    'xargs into a shell': ('FROM golang:1.23\nRUN echo "go build ." | xargs -I{} sh -c {}\n', 1),
    'the image\'s own GOPATH': ('FROM golang:1.23\nRUN n=0; ' + LOOP + '\nENV GOPATH=/go GOMODCACHE="/go/pkg/mod"\nRUN go build .\n', 0),
    'a COPY of the source is no module cache': ('FROM golang:1.23 AS deps\nRUN n=0; ' + LOOP + '\nFROM golang:1.23\nCOPY --from=deps /app /app\nRUN go build .\n', 1),
    'an exit whatever the count': ('FROM golang:1.23\nRUN until go mod download; do n=$((n+1)); [ "$n" -ge 5 ]; exit 1; done\n', 1),
    'su -c of a fetch': ('FROM golang:1.23\nRUN su app -c "go mod download"\n', 1),
    'a build after the loop, by &&': ('FROM golang:1.23\nRUN n=0; ' + LOOP + ' && go build .\n', 0),
    'a here-document that writes a FROM': ('FROM golang:1.23\nRUN n=0; ' + LOOP + '\nRUN cat > /inner <<EOF\nFROM alpine\nEOF\nRUN go build .\n', 0),
    'a here-document whose last line goes on': ('FROM golang:1.23\nRUN cat > /run.sh <<\'EOF\'\nexec app \\\nEOF\nRUN go mod download\nRUN go build .\n', 2),
    'a here-document after a continued RUN': ('FROM golang:1.23\nRUN mkdir -p /x && \\\n    cat > /x/cfg <<EOF\nkey=1\nEOF\nRUN go build .\n', 1),
    'a loop in a pipeline': ('FROM golang:1.23\nRUN n=0; ' + LOOP + ' 2>&1 | tee /tmp/dl.log\nRUN go build .\n', 2),
    'a loop at the end of a pipeline': ('FROM golang:1.23\nRUN n=0; echo go | ' + LOOP + '\nRUN go build .\n', 2),
    'a loop in the background': ('FROM golang:1.23\nRUN n=0; ' + LOOP + ' & wait\nRUN go build .\n', 2),
    'a loop after a background job': ('FROM golang:1.23\nRUN sleep 1 & n=0; ' + LOOP + '\nRUN go build .\n', 0),
    'a quoted script piped into sh': ('FROM golang:1.23\nRUN echo "go mod download && go build ." | sh\n', 1),
    'an increment in a subshell': ('FROM golang:1.23\nRUN until go mod download; do ( n=$((n+1)) ); [ "$n" -lt 5 ] || exit 1; done\n', 1),
    'the module cache copied to another place': ('FROM golang:1.23 AS deps\nRUN n=0; ' + LOOP + '\nFROM golang:1.23\n'
                                                 'COPY --from=deps /go/pkg/mod /root/go/pkg/mod\nRUN go build .\n', 1),
    'GOMODCACHE moved after the loop': ('FROM golang:1.23\nRUN n=0; ' + LOOP + '\nENV GOMODCACHE=/cache\nRUN go build .\n', 1),
    'go list in backticks': ('FROM golang:1.23\nRUN echo `go list -m all`\n', 1),
    'go mod why and graph': ('FROM golang:1.23\nRUN go mod why x\nFROM golang:1.23\nRUN go mod graph\n', 2),
    'a test before the loop is no exec form': ('FROM golang:1.23\nRUN [ -f go.mod ] && n=0 && ' + LOOP + '\nRUN go build .\n', 0),
    'a message to stderr in the loop': ('FROM golang:1.23\nRUN n=0; until go mod download; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; >&2 echo "go mod download failed"; done\nRUN go build .\n', 0),
    'a then-block that says why': ('FROM golang:1.23\nRUN n=0; until go mod download; do n=$((n+1)); if [ "$n" -ge 5 ]; then echo give up; exit 1; fi; done\nRUN go build .\n', 0),
    'an || that exits whatever': ('FROM golang:1.23\nRUN until go mod download; do n=$((n+1)); [ "$n" -lt 5 ] || echo no; exit 1; done\n', 1),
    'a timeout on each try': ('FROM golang:1.23\nRUN n=0; until timeout 300 go mod download; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; done\nRUN go build .\n', 0),
    'the loop as the samples write it': ('FROM golang:1.23\nRUN n=0; until go mod download; do n=$((n+1)); [ "$n" -lt 5 ] || exit 1; '
                                         'echo "go mod download failed (attempt $n), retrying in $((n*5))s"; sleep $((n*5)); done\nRUN go build .\n', 0),
    'an echo of go after cd': ('FROM golang:1.23\nRUN cd /tmp && echo "go build next"\n', 0),
    'an echo of go after an assignment': ('FROM golang:1.23\nRUN X=1 echo "go build next"\n', 0),
    'a part of the module cache copied as the whole': ('FROM golang:1.23 AS deps\nRUN n=0; ' + LOOP + '\nFROM golang:1.23\n'
                                                       'COPY --from=deps /go/pkg/mod/github.com /go/pkg/mod\nRUN go build .\n', 1),
}


def main():
    if '--self-test' in sys.argv:
        bad = [f'{k}: {problems(k, t)}, want {want}' for k, (t, want) in SELF_TEST.items() if len(problems(k, t)) != want]
        if bad:
            print('check_go_fetch_retried.py self-test failed:\n  ' + '\n  '.join(bad))
            return 1
    files = subprocess.run(['git', 'ls-files'], capture_output=True, text=True, check=True).stdout.split()
    found = []
    for f in files:
        base = os.path.basename(f).lower()
        if not (base.startswith('dockerfile') or base.startswith('containerfile') or base.endswith('.dockerfile')):
            continue
        with open(f, encoding='utf-8', errors='replace') as fh:
            found += problems(f, fh.read())
    for p in found:
        print(p)
    if found:
        print(f'{len(found)} Go module fetch(es) in Dockerfiles with no retry: see the top of {sys.argv[0]}')
        return 1
    print('every Go module fetch in the Dockerfiles runs in a retry loop')
    return 0


if __name__ == '__main__':
    sys.exit(main())
