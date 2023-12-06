## S3-Keploy

A simple CRUD application to showcase Keploy integration capabilities using [Go-Fiber](https://gofiber.io/) and [S3](https://aws.amazon.com/s3/)

## Prerequisites

1. [Go](https://go.dev/doc/install)
2. [AWS Access Key and Security Key](https://aws.github.io/aws-sdk-go-v2/docs/getting-started/#get-your-aws-access-keys)

## Running app on Ubuntu 22.04.03 LTS

### Setting aws credentials

Go to home directory      
Create `.aws` folder        
Inside `.aws` folder, create `credentials` name file        
Open `credentials` in any text editor and add following :       

```
[default]
aws_access_key_id = <YOUR_ACCESS_KEY_ID>
aws_secret_access_key = <YOUR_SECRET_ACCESS_KEY>
```
### Setting up application

```
git clone https://github.com/keploy/samples-go && cd S3-Keploy
go mod download
```

### Capture the Testcases

```shell
sudo -E env PATH="$PATH" keploy record -c 'go run .' 
```

#### Routes
- `/list` : GET - Get all buckets name
- `/getallobjects?bucket=<ENTER BUCKET NAME>` : GET - Get all objects name
- `/create` : POST - Create a new bucket
- `/upload?bucket=<ENTER BUCKET NAME>` : POST - Upload a file
- `/delete?bucket=<ENTER BUCKET NAME>` : DELETE - Delete a bucket
- `/deleteallobjects?bucket=<ENTER BUCKET NAME>` : DELETE - Delete all objects
- `/replacefile?bucket=<ENTER BUCKET NAME>` : PUT - Replace already present file

**Create a new bucket**
![image](https://github.com/rohitkbc/S3-Keploy/assets/100275369/e30fa3b6-78e8-4917-88b2-ebcec31736fb)

***Get all buckets name***
![image](https://github.com/rohitkbc/S3-Keploy/assets/100275369/f46fdc24-51bf-42dd-95c9-f0dbea235311)

***Upload a file***
![image](https://github.com/rohitkbc/S3-Keploy/assets/100275369/496ae0e3-99ae-43e2-b61b-24762e91b6bc)

***Replace already present file***
![image](https://github.com/rohitkbc/S3-Keploy/assets/100275369/e4b491fb-be1e-4849-a9d5-4ca2de1f5430)

***Delete a bucket***
![image](https://github.com/rohitkbc/S3-Keploy/assets/100275369/98339b6f-d95d-4009-9636-978b5496274f)

Once done, you can see the Test Cases on the Keploy server, like this:

![image](https://github.com/rohitkbc/S3-Keploy/assets/100275369/e1c3d469-11d0-430e-a47e-25cd59c6789e)

### Generate Test Runs

Now that we have our testcase captured, run the test file.
```shell
sudo -E env PATH=$PATH keploy test -c "go run ." --delay 20
```

Once done, you can see the Test Runs on the Keploy server, like this:

![image](https://github.com/rohitkbc/S3-Keploy/assets/100275369/a031a10c-e241-4a0d-9679-3fd44aa783c4)

### If you like the sample application, Don't forget to star us ✨

