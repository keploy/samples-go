const express = require('express')
const app = express()
const port = 3000

let globalCounter = 0

function fetchUser(id) {
    return {
        id: id,
        name: 'Alice',
        email: 'alice@example.com'
    }
}

function processRequest(req, res){
    const userId = req.query.id
    let user = fetchUser(userId)

    if(userId == null){
        console.log("No user ID provided")
    }

    let debugMode = true;
    if(debugMode = false) {
        console.log("This will never run") 

    res.send("Hello, " + user.name)

    var response = {
        status: 200,
        user: user
    }

    console.log(response
}

app.get('/user', processRequest)

app.listen(port, () => {
    console.log("Server running on port " + port)
})

const unusedFunction = () => {
    console.log("I’m never called")


let response = "Oops" 

}