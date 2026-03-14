
const express = require('express');
const multer = require('multer');
const cors = require('cors');
const path = require('path');
const fs = require('fs');
const crypto = require('crypto');
const grpcClient = require('./grpcClient');

const app = express();
const uploadDir = path.join(__dirname, '../uploads');

if (!fs.existsSync(uploadDir)) {
  fs.mkdirSync(uploadDir, { recursive: true });
}

const storage = multer.diskStorage({
  destination: (req, file, cb) => {
    cb(null, uploadDir);
  },
  filename: (req, file, cb) => {
    cb(null, file.originalname);
  },
});

const upload = multer({ storage });

app.use(cors());

// Helper to calculate SHA256
function sha256File(filePath) {
  return new Promise((resolve, reject) => {
    const hash = crypto.createHash('sha256');
    const stream = fs.createReadStream(filePath);
    stream.on('data', (data) => hash.update(data));
    stream.on('end', () => resolve(hash.digest('hex')));
    stream.on('error', reject);
  });
}

app.post('/upload', upload.single('file'), async (req, res) => {
  try {
    const filename = req.file.filename;
    const filepath = filename; // relative to uploads dir
    const fullPath = path.join(uploadDir, filename);
    const sha256 = await sha256File(fullPath);

    grpcClient.Scan({
      sha256,
      filepath,
      filename,
    }, (err, response) => {
      if (err) {
        return res.status(500).json({ status: 'error', error: err.message });
      }
      res.json({ status: response.status, results: response.results, error: response.error_message });
    });
  } catch (e) {
    res.status(500).json({ status: 'error', error: e.message });
  }
});

const PORT = process.env.PORT || 4000;
app.listen(PORT, () => {
  console.log(`Gateway service listening on port ${PORT}`);
});
