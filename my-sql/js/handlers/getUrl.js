const mysql = require('mysql2/promise');
const crypto = require('crypto');
const base58 = require('base58');

// Database connection details
const connectionDetails = {
    host: 'localhost',
    user: 'user',
    password: 'password',
    database: 'shorturl_db'
};

// Create a connection pool for better performance
const pool = mysql.createPool(connectionDetails);

// Generate a shortened URL
const generateShortLink = (initialLink) => {
    const hash = crypto.createHash('sha256').update(initialLink).digest('hex');
    const urlHashBytes = Buffer.from(hash, 'hex');
    const encoded = base58.encode(urlHashBytes);
    return encoded.slice(0, 8);
};

module.exports = {
    // PUT request to create a new shortened URL
    postUrl: async (req, res) => {
        const originalUrl = req.body.url;
        if (!originalUrl) {
            return res.status(400).json({ error: 'Missing URL parameter' });
        }
        const id = generateShortLink(originalUrl);
        try {
            const [rows] = await pool.query('INSERT INTO url_map (id, redirect_url, created_at, updated_at) VALUES (?, ?, NOW(), NOW())', [id, originalUrl]);
            return res.status(200).json({ ts: Date.now(), url: `http://localhost:3000/${id}` });
        } catch (err) {
            console.error(err);
            return res.status(500).json({ error: 'Unable to shorten URL' });
        }
    },

    // DELETE request to remove a shortened URL
    deleteUrl: async (req, res) => {
        const id = req.params.id;
        try {
            const [rows] = await pool.query('DELETE FROM url_map WHERE id = ?', [id]);
            return res.status(200).json({ rowsAffected: rows.affectedRows });
        } catch (err) {
            console.error(err);
            return res.status(500).json({ error: 'Error encountered while attempting to delete URL' });
        }
    },

    // GET request to retrieve and redirect to the original URL
    getUrl: async (req, res) => {
        const id = req.params.id;
        try {
            const [rows] = await pool.query('SELECT * FROM url_map WHERE id = ?', [id]);
            if (rows.length === 0) {
                return res.status(404).json({ error: 'Invalid URL ID' });
            }
            const entry = rows[0];
            return res.redirect(entry.redirect_url);
        } catch (err) {
            console.error(err);
            return res.status(500).json({ error: 'Error encountered while attempting to lookup URL' });
        }
    },

    // POST (or you can use PATCH) request to update a shortened URL
    updateUrl: async (req, res) => {
        const id = req.params.id;
        const newUrl = req.body.url;
        if (!newUrl) {
            return res.status(400).json({ error: 'Missing URL parameter' });
        }
        try {
            const [rows] = await pool.query('UPDATE url_map SET redirect_url = ?, updated_at = NOW() WHERE id = ?', [newUrl, id]);
            return res.status(200).json({ message: 'URL updated successfully' });
        } catch (err) {
            console.error(err);
            return res.status(500).json({ error: 'Could not update URL' });
        }
    }
};
