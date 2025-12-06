// Test data generation helpers
// Provides realistic test data for load tests

import { randomString } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

// Counter for unique data generation
let userCounter = 0;
let imageCounter = 0;

// Generate a unique username
export function generateUsername(prefix = 'loadtest') {
  userCounter++;
  const timestamp = Date.now();
  const random = randomString(6, 'abcdefghijklmnopqrstuvwxyz0123456789');
  return `${prefix}_${timestamp}_${random}`;
}

// Generate a unique email address
export function generateEmail(prefix = 'loadtest') {
  userCounter++;
  const timestamp = Date.now();
  const random = randomString(8, 'abcdefghijklmnopqrstuvwxyz0123456789');
  return `${prefix}_${timestamp}_${random}@example.com`;
}

// Generate a secure password
export function generatePassword(length = 16) {
  const charset = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*';
  let password = '';
  for (let i = 0; i < length; i++) {
    password += charset.charAt(Math.floor(Math.random() * charset.length));
  }
  // Ensure it meets minimum requirements (12 chars with complexity)
  return password.length >= 12 ? password : generatePassword(16);
}

// Generate image title
export function generateImageTitle() {
  imageCounter++;
  const adjectives = ['Beautiful', 'Stunning', 'Amazing', 'Gorgeous', 'Spectacular', 'Breathtaking'];
  const subjects = ['Sunset', 'Landscape', 'Portrait', 'Mountain', 'Ocean', 'Forest', 'City'];
  const locations = ['in Nature', 'at Dawn', 'at Dusk', 'in Summer', 'in Winter', 'at Night'];

  const adj = adjectives[Math.floor(Math.random() * adjectives.length)];
  const subj = subjects[Math.floor(Math.random() * subjects.length)];
  const loc = locations[Math.floor(Math.random() * locations.length)];

  return `${adj} ${subj} ${loc} #${imageCounter}`;
}

// Generate image description
export function generateImageDescription() {
  const templates = [
    'Captured this amazing view during my recent trip. Love the colors!',
    'One of my favorite shots from this location. What do you think?',
    'The lighting was perfect for this shot. So happy with how it turned out.',
    'Just experimenting with my new camera. Feedback welcome!',
    'Nature at its finest. Can\'t believe this is real.',
  ];
  return templates[Math.floor(Math.random() * templates.length)];
}

// Generate comment content
export function generateComment() {
  const comments = [
    'Beautiful shot! Love the composition.',
    'Amazing colors!',
    'This is stunning! Where was this taken?',
    'Great work! Keep it up.',
    'Wow, this is incredible!',
    'Love this! The lighting is perfect.',
    'Fantastic capture!',
    'This is so beautiful!',
    'Great photo! Very inspiring.',
    'Absolutely gorgeous!',
  ];
  return comments[Math.floor(Math.random() * comments.length)];
}

// Generate random tags (1-5 tags)
export function generateTags(count = null) {
  const allTags = [
    'nature', 'landscape', 'sunset', 'sunrise', 'ocean', 'mountain', 'forest',
    'city', 'portrait', 'travel', 'photography', 'art', 'beautiful', 'amazing',
    'summer', 'winter', 'spring', 'autumn', 'night', 'day', 'black-and-white',
    'color', 'vintage', 'modern', 'abstract', 'architecture', 'wildlife',
    'macro', 'street', 'documentary'
  ];

  const numTags = count || Math.floor(Math.random() * 5) + 1; // 1-5 tags
  const tags = [];
  const shuffled = allTags.sort(() => 0.5 - Math.random());

  for (let i = 0; i < Math.min(numTags, shuffled.length); i++) {
    tags.push(shuffled[i]);
  }

  return tags;
}

// Generate visibility (weighted towards public for realistic traffic)
export function generateVisibility() {
  const rand = Math.random();
  if (rand < 0.7) return 'public'; // 70% public
  if (rand < 0.9) return 'private'; // 20% private
  return 'unlisted'; // 10% unlisted
}

// Generate album title
export function generateAlbumTitle() {
  const templates = [
    'Summer Vacation 2024',
    'Travel Photos',
    'Nature Collection',
    'City Life',
    'Portrait Series',
    'Best of 2024',
    'Landscape Photography',
    'Wildlife Adventures',
  ];
  return templates[Math.floor(Math.random() * templates.length)] + ` ${randomString(4)}`;
}

// Generate album description
export function generateAlbumDescription() {
  const templates = [
    'A collection of my favorite photos from this series.',
    'Photos from my recent trip. Hope you enjoy!',
    'Curated selection of my best work.',
    'Various shots from different locations.',
    'A journey through beautiful landscapes.',
  ];
  return templates[Math.floor(Math.random() * templates.length)];
}

// Generate a small test image (base64 encoded)
// This is a minimal 1x1 pixel JPEG for testing purposes
export function generateTestImageData() {
  // Minimal valid JPEG (1x1 pixel, red)
  const jpegData = new Uint8Array([
    0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01,
    0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x43,
    0x00, 0x08, 0x06, 0x06, 0x07, 0x06, 0x05, 0x08, 0x07, 0x07, 0x07, 0x09,
    0x09, 0x08, 0x0A, 0x0C, 0x14, 0x0D, 0x0C, 0x0B, 0x0B, 0x0C, 0x19, 0x12,
    0x13, 0x0F, 0x14, 0x1D, 0x1A, 0x1F, 0x1E, 0x1D, 0x1A, 0x1C, 0x1C, 0x20,
    0x24, 0x2E, 0x27, 0x20, 0x22, 0x2C, 0x23, 0x1C, 0x1C, 0x28, 0x37, 0x29,
    0x2C, 0x30, 0x31, 0x34, 0x34, 0x34, 0x1F, 0x27, 0x39, 0x3D, 0x38, 0x32,
    0x3C, 0x2E, 0x33, 0x34, 0x32, 0xFF, 0xC0, 0x00, 0x0B, 0x08, 0x00, 0x01,
    0x00, 0x01, 0x01, 0x01, 0x11, 0x00, 0xFF, 0xC4, 0x00, 0x14, 0x00, 0x01,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x03, 0xFF, 0xC4, 0x00, 0x14, 0x10, 0x01, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0xFF, 0xDA, 0x00, 0x08, 0x01, 0x01, 0x00, 0x00, 0x3F, 0x00,
    0xFE, 0x8A, 0x28, 0xFF, 0xD9
  ]);

  return jpegData;
}

// Helper to create multipart form data for image upload
export function createImageFormData(imageData, metadata = {}) {
  const boundary = '----WebKitFormBoundary' + randomString(16);
  let body = '';

  // Add image file
  body += `--${boundary}\r\n`;
  body += 'Content-Disposition: form-data; name="file"; filename="test-image.jpg"\r\n';
  body += 'Content-Type: image/jpeg\r\n\r\n';
  body += String.fromCharCode.apply(null, imageData);
  body += '\r\n';

  // Add metadata fields
  if (metadata.title) {
    body += `--${boundary}\r\n`;
    body += 'Content-Disposition: form-data; name="title"\r\n\r\n';
    body += metadata.title + '\r\n';
  }

  if (metadata.description) {
    body += `--${boundary}\r\n`;
    body += 'Content-Disposition: form-data; name="description"\r\n\r\n';
    body += metadata.description + '\r\n';
  }

  if (metadata.visibility) {
    body += `--${boundary}\r\n`;
    body += 'Content-Disposition: form-data; name="visibility"\r\n\r\n';
    body += metadata.visibility + '\r\n';
  }

  if (metadata.tags) {
    body += `--${boundary}\r\n`;
    body += 'Content-Disposition: form-data; name="tags"\r\n\r\n';
    body += (Array.isArray(metadata.tags) ? metadata.tags.join(',') : metadata.tags) + '\r\n';
  }

  if (metadata.album_id) {
    body += `--${boundary}\r\n`;
    body += 'Content-Disposition: form-data; name="album_id"\r\n\r\n';
    body += metadata.album_id + '\r\n';
  }

  body += `--${boundary}--\r\n`;

  return {
    body: body,
    contentType: `multipart/form-data; boundary=${boundary}`
  };
}

// Utility to pick random item from array
export function randomItem(array) {
  return array[Math.floor(Math.random() * array.length)];
}
