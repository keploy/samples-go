<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0"/>
  <title>Gin App + Keploy + LocalStack Setup</title>
  <style>
    body {
      font-family: 'Segoe UI', sans-serif;
      margin: 40px;
      background-color: #f9fafb;
      color: #1f2937;
    }
    h1, h2, h3 {
      color: #111827;
    }
    h1 {
      font-size: 2rem;
    }
    h2 {
      margin-top: 30px;
      font-size: 1.5rem;
      border-bottom: 2px solid #d1d5db;
      padding-bottom: 5px;
    }
    pre {
      background-color: #1e293b;
      color: #f8fafc;
      padding: 1em;
      border-radius: 8px;
      overflow-x: auto;
      font-size: 0.95em;
    }
    code {
      background-color: #e5e7eb;
      padding: 2px 6px;
      border-radius: 4px;
    }
    p {
      margin-bottom: 10px;
    }
    a {
      color: #2563eb;
    }
    hr {
      border: none;
      height: 1px;
      background-color: #d1d5db;
      margin: 30px 0;
    }
  </style>
</head>
<body>

  <h1>📦 Gin App + Keploy + LocalStack Setup</h1>
  <p>This guide walks you through setting up your development environment using a Go Gin app, <a href="https://keploy.io" target="_blank">Keploy</a>, and <a href="https://localstack.cloud" target="_blank">LocalStack</a>. You’ll record HTTP + SQS interactions and run automated tests effortlessly.</p>

  <hr/>

  <h2>⚙️ 1. Install LocalStack</h2>
  <p><strong>Recommended (via pipx):</strong></p>
  <p><em>pipx keeps packages isolated and avoids dependency conflicts.</em></p>
  <pre><code>pip install --user pipx
pipx install localstack</code></pre>

  <p><strong>Alternative (via pip):</strong></p>
  <pre><code>pip install localstack</code></pre>

  <p><strong>Start LocalStack:</strong></p>
  <pre><code>localstack start -d (this flag is used to run it in background)</code></pre>
  <p><em>This will boot LocalStack and simulate AWS services locally.</em></p>

  <h2>🔐 2. Set Up LocalStack Auth Token</h2>
  <p>Get your token from <a href="https://app.localstack.cloud" target="_blank">LocalStack Console</a> and set it:</p>
  <pre><code>localstack config set auth_token &lt;your-token-here&gt;</code></pre>

  <h2>☁️ 3. Install AWS CLI & awslocal</h2>

  <h3>📦 On Linux (Debian/Ubuntu)</h3>
  <pre><code>sudo apt update
sudo apt install unzip curl -y

curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip
sudo ./aws/install</code></pre>

  <h3>🍎 On macOS (with Homebrew)</h3>
  <pre><code>brew install awscli</code></pre>

  <h3>🪟 On Windows</h3>
  <p><a href="https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html" target="_blank">Download and install AWS CLI</a></p>

  <h3>🧩 Install <code>awslocal</code> (universal)</h3>
  <pre><code>pip install awscli-local</code></pre>
  <p><em>awslocal is a wrapper for AWS CLI that redirects commands to LocalStack.</em></p>

  <h2>🔨 4. Build the Gin Application</h2>
  <pre><code>go build -o ginApp .</code></pre>
  <p><em>This compiles your Gin-based Go app into an executable named <code>ginApp</code>.</em></p>

  <h2>🐳 5. Start MongoDB with Docker</h2>
  <p><em>Create a Docker network and run MongoDB container on it.</em></p>
  <pre><code>docker network create keploy-network || true

docker run -p 27017:27017 --rm \
  --network keploy-network \
  --name mongoDb mongo</code></pre>

  <h2>🎥 6. Record Traffic Using Keploy</h2>
  <p><em>This starts Keploy in record mode to capture incoming requests.</em></p>
  <pre><code>sudo -E env PATH=$PATH /usr/local/bin/keploy record -c "./ginApp"</code></pre>

  <h2>🧪 7. Trigger Requests to Generate Test Cases</h2>
  <p><strong>POST request:</strong> Adds a new URL mapping.</p>
  <pre><code>curl --request POST \
  --url http://localhost:8080/url \
  --header 'Content-Type: application/json' \
  --data '{
  "url": "https://google.com"
}'</code></pre>

  <p><strong>GET request:</strong> Fetches a shortened URL.</p>
  <pre><code>curl --request GET http://localhost:8080/Lhr4BWAi</code></pre>

  <h2>📬 8. Send Message to Local SQS Queue</h2>
  <p><em>Push a message to the simulated SQS service in LocalStack.</em></p>
  <pre><code>awslocal sqs send-message \
  --queue-url http://sqs.us-east-1.localhost.localstack.cloud:4566/000000000000/localstack-queue \
  --message-body "Hello World"</code></pre>

  <h2>✅ 9. Run Tests with Keploy</h2>
  <p><em>Switch Keploy to test mode to replay requests and validate responses.</em></p>
  <pre><code>sudo -E env PATH=$PATH /usr/local/bin/keploy test -c "./ginApp" --delay 20 &gt; logs.txt</code></pre>

  <p><strong>Check test logs:</strong></p>
  <pre><code>cat logs.txt</code></pre>

  <h2>🙌 You're All Set!</h2>
  <p>You’ve successfully:</p>
  <ul>
    <li>Built and served a Gin application</li>
    <li>Recorded real traffic using Keploy</li>
    <li>Simulated SQS with LocalStack</li>
    <li>Automatically generated and executed tests 🎉</li>
  </ul>

</body>
</html>
