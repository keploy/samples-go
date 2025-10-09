# Body tests
curl http://localhost:8080/users-low-risk
curl http://localhost:8080/users-medium-risk
curl http://localhost:8080/users-medium-risk-with-addition
curl http://localhost:8080/users-high-risk-type
curl http://localhost:8080/users-high-risk-removal

# Status and Header tests
curl http://localhost:8080/status-change-high-risk
curl http://localhost:8080/content-type-change-high-risk
curl http://localhost:8080/header-change-medium-risk
curl http://localhost:8080/noisy-header
