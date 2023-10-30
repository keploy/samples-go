package constants

const VerificationMailTemplate = `<html>
	<head>
	  <title>Href Attribute Example</title>
	</head>
	<body>
	  <h1>Href Attribute Example</h1>
	  <p>
		<a href="https://www.freecodecamp.org/contribute/">The freeCodeCamp Contribution Page</a> shows you how and where you can contribute to freeCodeCamp's community and growth.
	  </p>
	  <p>
		Your verification code is: %d
	  </p>
	</body>
</html>`

const VerifyYourself = "Verify yourself with Influenza"

const API_SUCCESS_STATUS = "Success"

const API_FAILED_STATUS = "Failed"

const ApiFailStatus = "Fail"

var INVALID_TOKEN_RESPONSE = map[string]interface{}{
	"status":  ApiFailStatus,
	"message": "Invalid or No Token",
}

var INVALID_SUPER_ADMIN = map[string]interface{}{
	"status":  ApiFailStatus,
	"message": "Go back son",
}
