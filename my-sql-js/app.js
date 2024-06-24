const express = require('express');
const handlers = require('./handlers/getUrl');

const app = express();

app.use(express.json());
app.put('/:id', handlers.updateUrl);
app.delete('/:id', handlers.deleteUrl);
app.get('/:id', handlers.getUrl);
app.post('/url', handlers.postUrl); // or app.patch('/:id', handlers.updateUrl);

app.listen(3000, () => console.log('Server running on http://localhost:3000'));
