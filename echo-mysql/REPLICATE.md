sudo docker run --name mysql-container -e MYSQL_ROOT_PASSWORD=password -e MYSQL_DATABASE=uss -p 3306:3306 --rm mysql:latest
sudo -E env PATH=$PATH keploy record -c "./echo-mysql"
curl -X POST http://localhost:9090/seed
curl http://localhost:9090/query/active