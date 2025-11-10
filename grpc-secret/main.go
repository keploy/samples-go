package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sort"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	secretspb "grpc-secrets-sample/secrets"

	"google.golang.org/grpc/reflection"
)

// ---------- Deterministic helpers ----------

const (
	alnum          = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	hexChars       = "0123456789abcdef"
	b64Chars       = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789+/"
	fixedTimestamp = "2025-01-01T00:00:00Z"

	commonSeed     int64 = 20250909
	uniqueBaseSeed int64 = 777000

	jwtSeed  int64 = 424242
	curlSeed int64 = 515151
	cdnSeed  int64 = 616161

	metadataErrorMsg = "Failed to set response metadata: %v"
)

// Random generators (similar to Python version)

type secretGen struct {
	name string
	fn   func(r *rand.Rand) string
}

func randString(r *rand.Rand, charset string, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}

func awsAccessKeyID(r *rand.Rand) string {
	const upperDigits = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	return "AKIA" + randString(r, upperDigits, 16)
}

func awsSecretAccessKey(r *rand.Rand) string {
	return randString(r, b64Chars, 40)
}

func githubPAT(r *rand.Rand) string {
	return "ghp_" + randString(r, alnum, 36)
}

func slackWebhook(r *rand.Rand) string {
	upperDigits := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	return fmt.Sprintf(
		"https://hooks.slack.com/services/%s/%s/%s",
		randString(r, upperDigits, 9),
		randString(r, upperDigits, 9),
		randString(r, alnum, 24),
	)
}

func stripeLiveKey(r *rand.Rand) string {
	return "sk_live_" + randString(r, alnum, 28)
}

func googleAPIKey(r *rand.Rand) string {
	return "AIza" + randString(r, alnum+"_-", 35)
}

func twilioAuthToken(r *rand.Rand) string {
	return randString(r, hexChars, 32)
}

func sendgridAPIKey(r *rand.Rand) string {
	return "SG." + randString(r, b64Chars, 22) + "." + randString(r, b64Chars, 43)
}

func datadogAPIKey(r *rand.Rand) string {
	return randString(r, hexChars, 32)
}

func awsS3PresignedURL(r *rand.Rand) string {
	bucket := fmt.Sprintf("bucket-%s", randString(r, "abcdefghijklmnopqrstuvwxyz0123456789", 8))
	key := fmt.Sprintf("%s/%s.bin",
		randString(r, "abcdefghijklmnopqrstuvwxyz", 6),
		randString(r, alnum, 16),
	)
	xAmzCred := fmt.Sprintf("%s/%s/us-east-1/s3/aws4_request",
		randString(r, "0123456789", 8),
		randString(r, "0123456789", 8),
	)

	return fmt.Sprintf(
		"https://%s.s3.amazonaws.com/%s"+
			"?X-Amz-Algorithm=AWS4-HMAC-SHA256"+
			"&X-Amz-Credential=%s"+
			"&X-Amz-Date=20250101T000000Z"+
			"&X-Amz-Expires=900"+
			"&X-Amz-Signature=%s"+
			"&X-Amz-SignedHeaders=host",
		bucket,
		key,
		xAmzCred,
		randString(r, hexChars, 64),
	)
}

func azureStorageConnString(r *rand.Rand) string {
	return fmt.Sprintf(
		"DefaultEndpointsProtocol=https;"+
			"AccountName=%s;"+
			"AccountKey=%s;"+
			"EndpointSuffix=core.windows.net",
		randString(r, "abcdefghijklmnopqrstuvwxyz", 12),
		randString(r, b64Chars, 88),
	)
}

func mongoURI(r *rand.Rand) string {
	return fmt.Sprintf(
		"mongodb+srv://%s:%s@cluster%s.%s.mongodb.net/%s?retryWrites=true&w=majority&appName=%s",
		randString(r, "abcdefghijklmnopqrstuvwxyz", 6),
		randString(r, alnum, 16),
		randString(r, "abcdefghijklmnopqrstuvwxyz0123456789", 5),
		randString(r, "abcdefghijklmnopqrstuvwxyz", 6),
		randString(r, "abcdefghijklmnopqrstuvwxyz", 6),
		randString(r, alnum, 10),
	)
}

func postgresURL(r *rand.Rand) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s.%s.internal:5432/%s",
		randString(r, "abcdefghijklmnopqrstuvwxyz", 6),
		randString(r, alnum, 14),
		randString(r, "abcdefghijklmnopqrstuvwxyz", 6),
		randString(r, "abcdefghijklmnopqrstuvwxyz", 3),
		randString(r, "abcdefghijklmnopqrstuvwxyz", 6),
	)
}

func githubWebhookSecret(r *rand.Rand) string {
	return randString(r, alnum, 40)
}

func npmToken(r *rand.Rand) string {
	return "npm_" + randString(r, alnum, 36)
}

func gcpServiceAccountKeyID(r *rand.Rand) string {
	return randString(r, hexChars, 8)
}

func gcpServiceAccountEmail(r *rand.Rand) string {
	return fmt.Sprintf(
		"%s@%s.iam.gserviceaccount.com",
		randString(r, "abcdefghijklmnopqrstuvwxyz", 10),
		randString(r, "abcdefghijklmnopqrstuvwxyz", 8),
	)
}

func openaiAPIKey(r *rand.Rand) string {
	return "sk-" + randString(r, alnum+"_", 48)
}

func cloudflareAPIToken(r *rand.Rand) string {
	return randString(r, hexChars, 40)
}

func githubAppPrivateKeyID(r *rand.Rand) string {
	return randString(r, "0123456789", 6)
}

var generators = []secretGen{
	{"aws_access_key_id", awsAccessKeyID},
	{"aws_secret_access_key", awsSecretAccessKey},
	{"aws_s3_presigned_url", awsS3PresignedURL},
	{"github_pat", githubPAT},
	{"github_webhook_secret", githubWebhookSecret},
	{"slack_webhook_url", slackWebhook},
	{"stripe_live_key", stripeLiveKey},
	{"google_api_key", googleAPIKey},
	{"twilio_auth_token", twilioAuthToken},
	{"sendgrid_api_key", sendgridAPIKey},
	{"datadog_api_key", datadogAPIKey},
	{"azure_storage_connection_string", azureStorageConnString},
	{"mongodb_uri", mongoURI},
	{"postgres_url", postgresURL},
	{"npm_token", npmToken},
	{"gcp_service_account_key_id", gcpServiceAccountKeyID},
	{"gcp_service_account_email", gcpServiceAccountEmail},
	{"openai_api_key", openaiAPIKey},
	{"cloudflare_api_token", cloudflareAPIToken},
	{"github_app_private_key_id", githubAppPrivateKeyID},
}

func generateCommonSecrets() map[string]string {
	r := rand.New(rand.NewSource(commonSeed))
	out := make(map[string]string, len(generators))
	for _, g := range generators {
		out[g.name] = g.fn(r)
	}
	return out
}

func generateUniqueSecrets(endpointIdx int32) map[string]string {
	seed := uniqueBaseSeed + int64(endpointIdx)*111
	r := rand.New(rand.NewSource(seed))
	out := make(map[string]string)
	i := 0
	for len(out) < 40 {
		g := generators[i%len(generators)]
		key := fmt.Sprintf("%s_%d", g.name, len(out)+1)
		out[key] = g.fn(r)
		i++
	}
	return out
}

func stableID(prefix string, seed int64, n int) string {
	r := rand.New(rand.NewSource(seed))
	return prefix + "_" + randString(r, alnum, n)
}

// ---------- JWT + misc helpers ----------

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type jwtPayload struct {
	Aud string `json:"aud"`
	Sub string `json:"sub"`
	Ts  string `json:"ts"`
}

func b64URLNoPad(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func fakeJWT(r *rand.Rand) string {
	h := jwtHeader{Alg: "HS256", Typ: "JWT"}
	p := jwtPayload{
		Aud: "keploy-tests",
		Sub: stableID("user", 3100, 8),
		Ts:  fixedTimestamp,
	}

	hb, _ := json.Marshal(h)
	pb, _ := json.Marshal(p)

	sigBytes := make([]byte, 32)
	for i := range sigBytes {
		sigBytes[i] = byte(r.Intn(256))
	}

	seg1 := b64URLNoPad(hb)
	seg2 := b64URLNoPad(pb)
	seg3 := b64URLNoPad(sigBytes)

	return seg1 + "." + seg2 + "." + seg3
}

func opaqueToken(r *rand.Rand, n int) string {
	return randString(r, alnum+"._~-", n)
}

func pctAmp(s string) string {
	return strings.ReplaceAll(s, "&", "%26")
}

// ---------- Builders for gRPC responses ----------

func buildSecretResponse(id int32) *secretspb.SecretResponse {
	if id <= 0 {
		id = 1
	}
	endpointName := fmt.Sprintf("secret%d", id)

	commons := generateCommonSecrets()
	uniques := generateUniqueSecrets(id)

	resp := &secretspb.SecretResponse{
		Status: 200,
		Reason: "OK",
		Meta: &secretspb.Meta{
			Endpoint:  endpointName,
			Timestamp: fixedTimestamp,
			Version:   "v1",
			Trace: &secretspb.Trace{
				Session: &secretspb.Session{
					Id:     stableID("sess", 1000+int64(id), 24),
					Labels: []string{"sensitive", "test-fixture", "synthetic"},
					Routing: &secretspb.Routing{
						Region:   "us-east-1",
						Fallback: false,
						Partitions: []*secretspb.Partition{
							{Name: "p0", Weight: 60},
							{Name: "p1", Weight: 40},
						},
					},
				},
			},
		},
		Data: &secretspb.Data{
			PageInfo: &secretspb.PageInfo{
				Id: stableID("pg", 2000+int64(id), 8),
				PageMeta: &secretspb.PageMeta{
					Type: "LIST",
					FloatingMeta: &secretspb.FloatingMeta{
						Stack:       "HORIZONTAL",
						Arrangement: "SPACE_BETWEEN",
					},
				},
				Name: fmt.Sprintf("Deep Secret Dump %s", strings.ToUpper(endpointName)),
				LayoutParams: &secretspb.LayoutParams{
					BackgroundColor: "#0B1221",
					PageLayout: []*secretspb.LayoutItem{
						{Type: "widget", Id: "w-1"},
						{Type: "widget", Id: "w-2"},
					},
					UsePageLayout: true,
				},
				SeoData: &secretspb.SeoData{
					SeoDesc: "Synthetic secrets for scanner testing",
					Tags:    []string{"gitleaks", "testing"},
				},
			},
			PageContent: &secretspb.PageContent{
				HeaderWidgets: []*secretspb.HeaderWidget{
					{
						Id:   101,
						Type: "BREADCRUMBS",
						Data: &secretspb.HeaderWidgetData{
							Breadcrumbs: []*secretspb.Breadcrumb{
								{Label: "Root"},
								{Label: endpointName},
							},
						},
					},
				},
				Widgets: []*secretspb.Widget{
					{
						Id:   "cfg-001",
						Type: "CONFIG",
						Data: &secretspb.WidgetData{
							Providers: &secretspb.Providers{
								Aws: &secretspb.AwsProviders{
									Iam: &secretspb.AwsIAM{
										AccessKeyId:     commons["aws_access_key_id"],
										SecretAccessKey: commons["aws_secret_access_key"],
									},
									S3: &secretspb.AwsS3{
										PresignedExample: commons["aws_s3_presigned_url"],
									},
								},
								Github: &secretspb.GithubProviders{
									Pat:           commons["github_pat"],
									WebhookSecret: commons["github_webhook_secret"],
								},
								Slack: &secretspb.SlackProviders{
									WebhookUrl: commons["slack_webhook_url"],
								},
								Stripe: &secretspb.StripeProviders{
									LiveKey: commons["stripe_live_key"],
								},
								Google: &secretspb.GoogleProviders{
									ApiKey: commons["google_api_key"],
								},
								Databases: &secretspb.Databases{
									Mongo: &secretspb.MongoDB{
										Uri: commons["mongodb_uri"],
									},
									Postgres: &secretspb.PostgresDB{
										Url: commons["postgres_url"],
									},
								},
								Cloud: &secretspb.Cloud{
									Azure: &secretspb.AzureCloud{
										StorageConnectionString: commons["azure_storage_connection_string"],
									},
									Gcp: &secretspb.GcpCloud{
										ServiceAccount: &secretspb.GcpServiceAccount{
											KeyId: commons["gcp_service_account_key_id"],
											Email: commons["gcp_service_account_email"],
										},
									},
								},
								Ml: &secretspb.Ml{
									Openai: &secretspb.OpenAI{
										ApiKey: commons["openai_api_key"],
									},
								},
								Observability: &secretspb.Observability{
									Datadog: &secretspb.Datadog{
										ApiKey: commons["datadog_api_key"],
									},
								},
								PackageMgr: &secretspb.PackageManager{
									Npm: &secretspb.Npm{
										Token: commons["npm_token"],
									},
								},
							},
						},
					},
					{
						Id:   "cfg-002",
						Type: "SECRETS_UNIQUE",
						Data: &secretspb.WidgetData{
							UniqueSecrets: buildUniqueSecrets(uniques),
						},
					},
				},
				FooterWidgets: []*secretspb.Widget{},
				FloatingWidgets: []*secretspb.FloatingWidget{
					{
						Id:   "flt-1",
						Type: "SECRETS_DUPLICATED_VIEW",
						Data: &secretspb.FloatingWidgetData{
							Shadow: &secretspb.DeepSecretsShadow{
								Layer: &secretspb.ShadowLayer{
									Mirror: &secretspb.ShadowMirror{
										Commons: &secretspb.ShadowCommons{
											Aws: &secretspb.ShadowAws{
												AccessKeyId:     commons["aws_access_key_id"],
												SecretAccessKey: commons["aws_secret_access_key"],
											},
											Github: &secretspb.ShadowGithub{
												Pat: commons["github_pat"],
											},
											Stripe: &secretspb.ShadowStripe{
												LiveKey: commons["stripe_live_key"],
											},
											Google: &secretspb.ShadowGoogle{
												ApiKey: commons["google_api_key"],
											},
										},
									},
								},
							},
						},
					},
				},
			},
			HasNudge: false,
			Ttl:      10,
		},
	}

	return resp
}

func buildUniqueSecrets(uniques map[string]string) *secretspb.UniqueSecrets {
	keys := make([]string, 0, len(uniques))
	for k := range uniques {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var bucketAEntries []*secretspb.Entry
	var bucketBEntries []*secretspb.Entry

	for i, k := range keys {
		e := &secretspb.Entry{
			Key:   k,
			Value: uniques[k],
		}
		if i < len(keys)/2 {
			bucketAEntries = append(bucketAEntries, e)
		} else {
			bucketBEntries = append(bucketBEntries, e)
		}
	}

	layer3 := &secretspb.SecretsLayer{
		Name: "layer-3",
		Buckets: []*secretspb.Bucket{
			{Name: "bucket-a", Entries: bucketAEntries},
			{Name: "bucket-b", Entries: bucketBEntries},
		},
	}

	layer2 := &secretspb.SecretsLayer{
		Name:     "layer-2",
		Children: []*secretspb.SecretsLayer{layer3},
	}

	layer1 := &secretspb.SecretsLayer{
		Name:     "layer-1",
		Children: []*secretspb.SecretsLayer{layer2},
	}

	return &secretspb.UniqueSecrets{
		Layers: []*secretspb.SecretsLayer{layer1},
	}
}

func buildAstroResponse() *secretspb.AstroResponse {
	// Based on ASTRO_JSON in your Python sample
	return &secretspb.AstroResponse{
		Status: 200,
		Reason: "OK",
		Data: &secretspb.AstroData{
			Catalog: &secretspb.Catalog{
				CatalogId: "NGC-ORION",
				Name:      "Deep Sky Catalog – Orion Region",
				Entries: []*secretspb.CatalogEntry{
					{
						Object: &secretspb.AstroObject{
							Id:   "M42",
							Type: "Nebula",
							Content: []*secretspb.AstroContent{
								{
									Language: 1,
									Desc: &secretspb.AstroDesc{
										Text: "\"\\u003cdiv style=\\\"text-align:justify\\\" \\u003eThe Orion Nebula (M42) is a diffuse nebula visible to the naked eye; it is one of the most studied regions of star formation.\\u003c\\/div\\u003e\\n\"",
									},
									Images: []*secretspb.AstroImage{
										{Alt: "\\\\frac{{L}}{{4\\pi d^{2}}}", Src: "luminosity_distance.png"},
										{Alt: "v_{esc}=\\\\sqrt{\\\\frac{2GM}{{R}}}", Src: "escape_velocity.png"},
									},
									ObjectLanguage: "ENGLISH",
									Nature:         "CATALOG_ENTRY",
								},
								{
									Language: 2,
									Desc: &secretspb.AstroDesc{
										Text: "\"\\u003cdiv style=\\\"text-align:justify\\\" \\u003e\\u0913\\u0930\\u093e\\u092f\\u0928 \\u0928\\u0947\\u092c\\u094d\\u092f\\u0942\\u0932\\u093e (M42) \\u090f\\u0915 \\u0935\\u093f\\u0938\\u094d\\u0924\\u0943\\u0924 \\u0928\\u0947\\u092c\\u094d\\u092f\\u0942\\u0932\\u093e \\u0939\\u0948 \\u091c\\u094b \\u0928\\u0902\\u0917\\u0940 \\u0906\\u0902\\u0916\\u094b\\u0902 \\u0938\\u0947 \\u0926\\u093f\\u0916\\u093e\\u0908 \\u0926\\u0947\\u0924\\u0940 \\u0939\\u0948।\\u003c\\/div\\u003e\\n\"",
									},
									Images: []*secretspb.AstroImage{
										{Alt: "\\\\int_0^{R} 4\\pi r^2 \\rho(r)\\,dr = {{M_{\\odot}}}", Src: "mass_integral.png"},
									},
									ObjectLanguage: "HINDI",
									Nature:         "CATALOG_ENTRY",
								},
							},
							Spectrum: &secretspb.Spectrum{
								WavelengthNm: []float64{486.1, 656.3},
								Lines:        []string{"H\\\\beta", "H\\\\alpha"},
							},
						},
						Metadata: &secretspb.AstroMetadata{
							Ra:         "05h35m17.3s",
							Dec:        "-05\u00B023'28\"",
							DistanceLy: 1344,
						},
					},
				},
			},
		},
	}
}

func buildJwtLabResponse() *secretspb.JwtLabResponse {
	r := rand.New(rand.NewSource(jwtSeed))
	j := fakeJWT(r)
	uid := stableID("user", 9090, 12)

	base := fmt.Sprintf("https://example.test/api/callback?token=%s&user_uuid=%s&mode=demo", j, uid)

	return &secretspb.JwtLabResponse{
		CaseName: "jwtlab",
		Status:   200,
		Meta: &secretspb.Meta{
			Endpoint:  "jwtlab",
			Timestamp: fixedTimestamp,
		},
		Examples: &secretspb.JwtExamples{
			UrlRaw:    base,
			UrlPctAmp: pctAmp(base),
			JsonParam: &secretspb.JwtJsonParam{
				Token: j,
			},
		},
	}
}

func buildCurlMixResponse() *secretspb.CurlMixResponse {
	commons := generateCommonSecrets()
	r := rand.New(rand.NewSource(curlSeed))

	bearer := opaqueToken(r, 40)
	apiKey := commons["openai_api_key"]

	curl := fmt.Sprintf(
		"curl -s -H 'Authorization: Bearer %s' -H 'X-Api-Key: %s' https://api.example.test/v1/things",
		bearer, apiKey,
	)

	return &secretspb.CurlMixResponse{
		CaseName: "curlmix",
		Status:   200,
		Meta: &secretspb.Meta{
			Endpoint:  "curlmix",
			Timestamp: fixedTimestamp,
		},
		Shadow: &secretspb.CurlShadow{
			BearerTokenShadow: bearer,
			ApiKeyShadow:      apiKey,
		},
		Curl: curl,
	}
}

func buildCdnResponse() *secretspb.CdnResponse {
	r := rand.New(rand.NewSource(cdnSeed))
	hmacHex := randString(r, hexChars, 64)
	hdntsPlain := fmt.Sprintf("hdnts=st=1700000000~exp=1999999999~acl=/*~hmac=%s", hmacHex)

	return &secretspb.CdnResponse{
		CaseName: "cdn",
		Status:   200,
		Meta: &secretspb.Meta{
			Endpoint:  "cdn",
			Timestamp: fixedTimestamp,
		},
		Urls: &secretspb.CdnUrls{
			AkamaiHdnts: fmt.Sprintf("https://cdn.example.test/asset.m3u8?%s", hdntsPlain),
		},
		Fields: &secretspb.CdnFields{
			HdntsPlain: hdntsPlain,
		},
	}
}

func buildHealthResponse() *secretspb.HealthResponse {
	return &secretspb.HealthResponse{
		Status: "ok",
		Ts:     fixedTimestamp,
	}
}

// ---------- Header/Metadata helpers ----------

func extractAuthFromMetadata(ctx context.Context) (jwt string, apiKey string, sessionToken string) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", "", ""
	}

	// Extract JWT from Authorization header
	if auth := md.Get("authorization"); len(auth) > 0 {
		jwt = auth[0]
		// Remove "Bearer " prefix if present
		jwt = strings.TrimPrefix(jwt, "Bearer ")
	}

	// Extract API key
	if key := md.Get("x-api-key"); len(key) > 0 {
		apiKey = key[0]
	}

	// Extract session token
	if session := md.Get("x-session-token"); len(session) > 0 {
		sessionToken = session[0]
	}

	return jwt, apiKey, sessionToken
}

func setResponseMetadata(ctx context.Context) error {
	// Generate secrets for response headers
	r := rand.New(rand.NewSource(jwtSeed + 100))
	responseJWT := fakeJWT(r)

	commons := generateCommonSecrets()

	// Create response metadata with secrets
	md := metadata.Pairs(
		"x-response-token", responseJWT,
		"x-api-secret", commons["openai_api_key"],
		"x-stripe-key", commons["stripe_live_key"],
		"x-github-token", commons["github_pat"],
		"x-aws-access-key", commons["aws_access_key_id"],
		"x-aws-secret", commons["aws_secret_access_key"],
		"x-session-id", stableID("sess", 7777, 32),
		"x-request-id", stableID("req", 8888, 16),
	)

	return grpc.SendHeader(ctx, md)
}

func logIncomingHeaders(method string, jwt, apiKey, sessionToken string) {
	log.Printf("[%s] Incoming headers:", method)
	if jwt != "" {
		log.Printf("  - Authorization (JWT): %s", jwt[:min(len(jwt), 50)]+"...")
	}
	if apiKey != "" {
		log.Printf("  - X-Api-Key: %s", apiKey)
	}
	if sessionToken != "" {
		log.Printf("  - X-Session-Token: %s", sessionToken)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ---------- gRPC server implementation ----------

type server struct {
	secretspb.UnimplementedSecretServiceServer
}

func (s *server) GetSecret(ctx context.Context, req *secretspb.SecretRequest) (*secretspb.SecretResponse, error) {
	// Extract and log incoming headers
	jwt, apiKey, sessionToken := extractAuthFromMetadata(ctx)
	logIncomingHeaders("GetSecret", jwt, apiKey, sessionToken)

	// Set response headers with secrets
	if err := setResponseMetadata(ctx); err != nil {
		log.Printf(metadataErrorMsg, err)
	}

	return buildSecretResponse(req.GetId()), nil
}

func (s *server) GetAstro(ctx context.Context, _ *secretspb.AstroRequest) (*secretspb.AstroResponse, error) {
	// Extract and log incoming headers
	jwt, apiKey, sessionToken := extractAuthFromMetadata(ctx)
	logIncomingHeaders("GetAstro", jwt, apiKey, sessionToken)

	// Set response headers with secrets
	if err := setResponseMetadata(ctx); err != nil {
		log.Printf(metadataErrorMsg, err)
	}

	return buildAstroResponse(), nil
}

func (s *server) JwtLab(ctx context.Context, _ *secretspb.JwtLabRequest) (*secretspb.JwtLabResponse, error) {
	// Extract and log incoming headers
	jwt, apiKey, sessionToken := extractAuthFromMetadata(ctx)
	logIncomingHeaders("JwtLab", jwt, apiKey, sessionToken)

	// Set response headers with secrets
	if err := setResponseMetadata(ctx); err != nil {
		log.Printf(metadataErrorMsg, err)
	}

	return buildJwtLabResponse(), nil
}

func (s *server) CurlMix(ctx context.Context, _ *secretspb.CurlMixRequest) (*secretspb.CurlMixResponse, error) {
	// Extract and log incoming headers
	jwt, apiKey, sessionToken := extractAuthFromMetadata(ctx)
	logIncomingHeaders("CurlMix", jwt, apiKey, sessionToken)

	// Set response headers with secrets
	if err := setResponseMetadata(ctx); err != nil {
		log.Printf(metadataErrorMsg, err)
	}

	return buildCurlMixResponse(), nil
}

func (s *server) Cdn(ctx context.Context, _ *secretspb.CdnRequest) (*secretspb.CdnResponse, error) {
	// Extract and log incoming headers
	jwt, apiKey, sessionToken := extractAuthFromMetadata(ctx)
	logIncomingHeaders("Cdn", jwt, apiKey, sessionToken)

	// Set response headers with secrets
	if err := setResponseMetadata(ctx); err != nil {
		log.Printf(metadataErrorMsg, err)
	}

	return buildCdnResponse(), nil
}

func (s *server) Health(ctx context.Context, _ *secretspb.HealthRequest) (*secretspb.HealthResponse, error) {
	// Extract and log incoming headers
	jwt, apiKey, sessionToken := extractAuthFromMetadata(ctx)
	logIncomingHeaders("Health", jwt, apiKey, sessionToken)

	// Set response headers with secrets
	if err := setResponseMetadata(ctx); err != nil {
		log.Printf(metadataErrorMsg, err)
	}

	return buildHealthResponse(), nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	secretspb.RegisterSecretServiceServer(grpcServer, &server{})

	// 👇 Add this line
	reflection.Register(grpcServer)

	log.Println("gRPC Secret Lab listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
