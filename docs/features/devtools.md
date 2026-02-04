# Developer Tools

GAGOS includes a suite of developer utilities for common encoding, hashing, and key management tasks.

## Base64

Encode and decode Base64 strings.

### Encode

1. Open Dev Tools window
2. Select "Base64" tab
3. Enter text in input field
4. Click "Encode"
5. Copy encoded result

### Decode

1. Enter Base64 string
2. Click "Decode"
3. View decoded text

### K8s Secret Decoder

Decode Kubernetes Secret values (which are Base64-encoded):

1. Paste the entire Secret YAML
2. Click "Decode Secret"
3. View decoded key-value pairs

---

## Hashing

Generate cryptographic hashes of text.

### Supported Algorithms

- **MD5** - 128-bit hash (not secure for passwords)
- **SHA1** - 160-bit hash (deprecated for security)
- **SHA256** - 256-bit hash (recommended)
- **SHA512** - 512-bit hash (strongest)

### Generate Hash

1. Open Dev Tools window
2. Select "Hash" tab
3. Enter text to hash
4. Click "Generate"
5. View all hash outputs

### Compare Hashes

Verify if two strings produce the same hash:

1. Enter first string
2. Enter second string (or known hash)
3. Click "Compare"
4. Results show match/mismatch

---

## Certificate Tools

Parse and validate SSL/TLS certificates.

### Parse Certificate

1. Open Dev Tools window
2. Select "Certificates" tab
3. Paste PEM-encoded certificate
4. Click "Parse"

**Information displayed:**
- Subject (CN, O, OU)
- Issuer
- Serial number
- Validity period (Not Before, Not After)
- Public key algorithm and size
- Signature algorithm
- Subject Alternative Names (SANs)
- Key usage

### Check Remote Certificate

1. Enter hostname
2. Click "Check"
3. View certificate details from live connection

---

## SSH Key Generator

Generate SSH key pairs for authentication.

### Key Types

- **RSA** - Traditional, widely compatible (2048, 4096 bits)
- **ECDSA** - Elliptic curve (256, 384, 521 bits)
- **ED25519** - Modern, fast, secure (recommended)

### Generate Key Pair

1. Open Dev Tools window
2. Select "SSH Keys" tab
3. Select key type
4. Select key size (for RSA/ECDSA)
5. Optional: Enter passphrase
6. Click "Generate"

**Output:**
- Private key (keep secret!)
- Public key (add to `~/.ssh/authorized_keys`)

### Validate SSH Key

Verify an existing SSH key:

1. Paste SSH key (public or private)
2. Click "Validate"
3. View key information

---

## JSON Tools

Format and minify JSON data.

### Format JSON

Pretty-print JSON with indentation:

1. Open Dev Tools window
2. Select "JSON" tab
3. Paste JSON
4. Click "Format"
5. Copy formatted output

### Minify JSON

Remove whitespace for compact JSON:

1. Paste JSON
2. Click "Minify"
3. Copy minified output

---

## Text Diff

Compare two text blocks side by side.

### Compare Text

1. Open Dev Tools window
2. Select "Diff" tab
3. Paste first text in left panel
4. Paste second text in right panel
5. Click "Compare"

**Output shows:**
- Added lines (green)
- Removed lines (red)
- Changed lines (yellow)
- Unchanged lines (white)

---

## API Reference

```bash
# Base64 Encode
curl -X POST http://localhost:8080/api/v1/devtools/base64/encode \
  -H "Content-Type: application/json" \
  -d '{"input":"Hello World"}'

# Base64 Decode
curl -X POST http://localhost:8080/api/v1/devtools/base64/decode \
  -H "Content-Type: application/json" \
  -d '{"input":"SGVsbG8gV29ybGQ="}'

# Generate Hashes
curl -X POST http://localhost:8080/api/v1/devtools/hash \
  -H "Content-Type: application/json" \
  -d '{"input":"password123"}'

# Parse Certificate
curl -X POST http://localhost:8080/api/v1/devtools/cert/parse \
  -H "Content-Type: application/json" \
  -d '{"pem":"-----BEGIN CERTIFICATE-----\n..."}'

# Generate SSH Key
curl -X POST http://localhost:8080/api/v1/devtools/ssh/generate \
  -H "Content-Type: application/json" \
  -d '{"type":"ed25519","passphrase":""}'

# Format JSON
curl -X POST http://localhost:8080/api/v1/devtools/json/format \
  -H "Content-Type: application/json" \
  -d '{"input":"{\"key\":\"value\"}"}'

# Text Diff
curl -X POST http://localhost:8080/api/v1/devtools/diff \
  -H "Content-Type: application/json" \
  -d '{"text1":"line1\nline2","text2":"line1\nline3"}'
```

## Use Cases

### Decoding Kubernetes Secrets

```bash
kubectl get secret my-secret -o yaml
```

Copy the YAML and use the K8s Secret Decoder to see actual values.

### Verifying File Integrity

Generate SHA256 hash of a file's contents and compare with published hash.

### Generating Deploy Keys

Use SSH Key Generator to create ED25519 keys for CI/CD systems.

### Debugging API Responses

Use JSON Format to make minified JSON readable.

### Comparing Configurations

Use Text Diff to compare old vs new config files.
