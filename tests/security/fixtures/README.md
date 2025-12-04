# Security Test Fixtures

This directory contains test files for security validation testing.

## Files

### eicar.txt
Standard EICAR antivirus test file. This is NOT actual malware - it's a standard test file recognized by all antivirus engines.
- Source: https://www.eicar.org/download-anti-malware-testfile/
- Purpose: Test ClamAV malware detection
- Safe to commit to repository

### clean_image.jpg
Minimal valid JPEG file for positive test cases.
- Purpose: Verify valid images pass validation
- Format: JPEG with valid magic bytes

### oversized.bin
11MB binary file (exceeds 10MB limit).
- Purpose: Test file size validation
- Generated: Random data

### fake_jpeg.jpg
Plain text file with .jpg extension (no valid image data).
- Purpose: Test MIME type validation by content
- Contains: Plain text, not image data

## Generating Fixtures

Run `make generate-test-fixtures` to regenerate these files (if needed).

## Security Note

All files in this directory are for testing purposes only. The EICAR test file is NOT malware and is safe to distribute and commit to version control.
