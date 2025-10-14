This is a simple Go web server that tells you dad jokes by fetching them from the `icanhazdadjoke.com` API.

This guide demonstrates how to use **Keploy** to generate test cases and data mocks from the actual API calls, allowing you to test your application without any external dependencies or mocks. 🧑‍💻

## 🧪 Testing with Native Go Binary (Linux)

This method involves running the Go application directly on your machine. Keploy will capture the outgoing API calls your application makes and automatically create mocks for them.

### 1. Prepare the Application

First, build the application binary.

```bash
# Initialize Go modules (if you haven't already)
go mod init go-joke-app
go mod tidy

# Build the executable
go build -o go-joke-app
```

### 2. Record Test Cases & Mocks

Now, run the `record` command. Keploy will start your application and monitor any API calls it makes.

```bash
sudo -E keploy record -c "./go-joke-app"
```

In a **separate terminal**, send some requests to your application to generate traffic. Each request will be captured by Keploy as a test case.

```bash
curl http://localhost:8080/joke
```

Once you're done, stop the Keploy process in the first terminal by pressing **`Ctrl + C`**. You'll see two new directories created:

  * `keploy/`: Contains the captured test cases and data mocks (`test-0.yml`, `mocks.yml`).
  * `reports/`: Contains test run reports.

### 3. Run the Tests

Now, run the `test` command. Keploy will start your application again, but this time it will **intercept the API calls** and serve the responses from the mocks it captured in the previous step. Your application is now being tested in complete isolation, without needing to connect to the internet. 🌐➡️📦

```bash
sudo -E keploy test -c "./go-joke-app" --delay 10
```

You'll see the test results in your terminal, showing which test cases passed or failed.

-----

## 🐳 Testing with Docker

This method runs your application inside a Docker container. Keploy instruments the container to capture and replay the API interactions.

### 1. Prepare the Docker Image

First, build the Docker image for your application.

```bash
# Initialize Go modules (if you haven't already)
go mod init go-joke-app
go mod tidy

# Build the Docker image
docker build -t go-joke-app:latest .
```

### 2. Record Test Cases & Mocks

Run the Keploy `record` command, pointing it to the `docker run` command for your application. Keploy will automatically create a network to intercept the traffic.

```bash
sudo -E keploy record -c "docker run --rm -p 8080:8080 --name go-joke-app-container --network=keploy-network go-joke-app:latest" --container-name go-joke-app-container
```

Just like before, open a **separate terminal** and send requests to generate traffic:

```bash
curl http://localhost:8080/joke
```

Stop the Keploy process with **`Ctrl + C`** when you're finished. The `keploy/` directory will be updated with your new test cases and mocks.

### 3. Run the Tests

Finally, run the `test` command. Keploy will run your application container along with a "mock" container that serves the captured API responses. This ensures your tests are fast, reliable, and independent of external services.

```bash
sudo -E keploy test -c "docker run --rm -p 8080:8080 --name go-joke-app-container --network=keploy-network go-joke-app:latest" --container-name go-joke-app-container --delay 10
```

The test run results will be displayed in your console. That's it\! 🎉