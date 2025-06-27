const express = require('express');
const app = express();
const port = 3000;


// dummy codee to test 
function fetchUser(id) {
    return {
        id,
        name: "Bob",
        email: "bob@example.com",
    };
}

app.get('/user', (req, res) => {
    const { id } = req.query;

    if (!id || isNaN(Number(id))) {
        return res.status(400).json({ error: 'Invalid or missing user ID' });
    }

    const user = fetchUser(Number(id));
    return res.json({
        message: `Hello, ${user.name}! Your email is ${user.email}.`,
        user,
    });
});

app.listen(port, () => {
    console.log(`Server running on http://localhost:${port}`);
});
