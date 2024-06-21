# Introduction
This is a simple url shortener written in Golang and MySQL

# Requirments to run 
1. Golang [How to install Golang](https://go.dev/doc/install)
2. Docker [How to install Docker?](https://docs.docker.com/engine/install/)

# Setting up the project
Run the following commands to setup the project

``` bash
git clone https://github.com/keploy/samples-go.git && cd samples-go/mux-mysql
go mod download
```

# Installing Keploy 
 
```  bash
curl -O https://raw.githubusercontent.com/keploy/keploy/main/keploy.sh && source keploy.sh 
```

# Running the Application
### Method 1 : Using natively installed Keploy
1. Spin up the mysql docker container
 
    ``` bash
    docker run -p 3306:3306 --rm --name mysql -e MYSQL_ROOT_PASSWORD=my-secret-pw -d mysql:latest
    ```
    
2. The MySQL Database takes some time to be ready after spinning up the container, you can wait a minute or just run the following command to monitor the logs

    ``` bash
    watch -n 1 docker logs mysql
    ```
    
3. Once we get the following message in the logs we are ready to start our golang server

   ``` bash
    2024-01-19 13:53:43+00:00 [Note] [Entrypoint]: Stopping temporary server
    2024-01-19T13:53:43.854798Z 10 [System] [MY-013172] [Server] Received SHUTDOWN from user root. Shutting down mysqld (Version: 8.2.0).
    2024-01-19T13:54:00.148602Z 0 [System] [MY-010910] [Server] /usr/sbin/mysqld: Shutdown complete (mysqld 8.2.0)  MySQL Community Server - GPL.
    ```
   
4. Run the following command to build the executable out of our golang code

    ``` bash
    export ConnectionString="root:my-secret-pw@tcp(localhost:3306)/mysql"
    
    go build -o main -cover
    ```
    
    If you get some error like this after running `./main`, 
    
    ``` bash
    [mysql] 2024/01/19 21:28:04 packets.go:37: unexpected EOF
    [mysql] 2024/01/19 21:28:04 packets.go:37: unexpected EOF
    [mysql] 2024/01/19 21:28:04 packets.go:37: unexpected EOF
    2024/01/19 21:28:04 Couldnt create store driver: bad connection
    ```

    This means that the mysql db is not ready yet, watch out for logs in **Step 2** and wait for 20-30s

    
5. To capture test cases for our application use the following command

   ``` bash
    keploy record -c "./main"
    ```
   ![Running Keploy](https://github.com/heyyakash/samples-go/assets/85030597/05644fa2-64eb-4cd2-8c08-0c1d0690528f)

   
   
6. To generate test cases, we can simply use cURL or Postman.

    ``` bash
    curl -X POST localhost:8080/create \ 
    -d '{"link":"https://google.com"}'
    ```
    
    This should return the following output

    ``` json
    {
        "message":"Converted",
        "link":"http://localhost:8080/link/1",
        "status":true
    }
    
    ```
    
7. To get all the links, run 

    ``` bash
    curl localhost:8080/all
    ```
    
8. To get redirected to the real url, just use the shortened link in browser or run

    ``` bash
    curl localhost:8080/links/1
    ```
    
9. All these test cases are captured by Keploy
   
![Screenshot from 2024-01-19 20-18-44](https://github.com/heyyakash/samples-go/assets/85030597/5b43b17b-81ef-453f-b843-cb9a7619d157)

#### Running the test cases
1. Just one simple command, run

   ``` bash
    keploy test -c "./main" --goCoverage
   ```
   
![Screenshot from 2024-01-19 20-21-49](https://github.com/heyyakash/samples-go/assets/85030597/8167df44-14ec-4037-a768-5e19f8a81826)

3. Done..


## Method 2 : Using Dockerized version
1. Spin up the mysql docker container

   ``` bash
    docker run -p 3306:3306 --rm --name mysql --network keploy-network -e MYSQL_ROOT_PASSWORD=my-secret-pw -d mysql:latest
    ```
   
2. The MySQL Database takes some time to be ready after spinning up the container, you can wait a minute or just run the following command to monitor the logs

   ``` bash
    watch -n 1 docker logs mysql
    ```
   
3. Once we get the following message in the logs we are ready to start our golang server

    ``` bash
    2024-01-19 13:53:43+00:00 [Note] [Entrypoint]: Stopping temporary server
    2024-01-19T13:53:43.854798Z 10 [System] [MY-013172] [Server] Received SHUTDOWN from user root. Shutting down mysqld (Version: 8.2.0).
    2024-01-19T13:54:00.148602Z 0 [System] [MY-010910] [Server] /usr/sbin/mysqld: Shutdown complete (mysqld 8.2.0)  MySQL Community Server - GPL.
    ```
    
4. Build the docker image of the application

   ``` bash
    docker build -t url-short .
    ```
    
5. To start the server with keploy, run

    ``` bash
    keploy record -c "docker run -p 8080:8080 --name urlshort --rm --network keploy-network url-short:latest"
    ```
    There you go, keploy has started capturing incomming requests

    ![Running Keploy](https://github.com/heyyakash/samples-go/assets/85030597/2b4f3c04-4631-4f9a-b317-7fdb6db87879)
    
7. Make some API Requests: - 
    
    ``` bash
    curl -X POST localhost:8080/create \ 
    -d '{"link":"https://google.com"}'
    ```
    
    This should return the following output

    ``` json
    {
        "message":"Converted",
        "link":"http://localhost:8080/link/1",
        "status":true
    }
    
    ```
    
    To get all the links, run 

    ``` bash
    curl localhost:8080/all
    ```

    ![Screenshot from 2024-01-19 20-36-09](https://github.com/heyyakash/samples-go/assets/85030597/eb17602d-c3cd-43e1-bf64-a55af62902f2)

8. To run the tests, run

    ```  bash
    keploy test -c "docker run -p 8080:8080 --name urlshort --rm --network keploy-network url-short:latest"
    ```

    ![Screenshot from 2024-01-19 20-38-08](https://github.com/heyyakash/samples-go/assets/85030597/472cab5e-9687-4fc5-bd57-3c52f56feedf)


