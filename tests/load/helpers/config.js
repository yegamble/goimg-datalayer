// Configuration helper for k6 load tests
// Centralizes environment configuration and base URLs

export const config = {
  // Base API URL (can be overridden via environment variable)
  baseURL: __ENV.API_BASE_URL || 'http://localhost:8080/api/v1',

  // Test configuration
  testData: {
    // Number of users to pre-create for tests
    userPoolSize: parseInt(__ENV.USER_POOL_SIZE) || 50,

    // Image upload configuration
    imageMaxSize: 2 * 1024 * 1024, // 2MB
    imageFormats: ['image/jpeg', 'image/png'],

    // Pagination defaults
    defaultPerPage: 20,
    maxPerPage: 100,
  },

  // Performance thresholds
  thresholds: {
    // Non-upload endpoints
    standardP95: 200, // ms
    standardP99: 500, // ms

    // Upload endpoints
    uploadP95: 5000, // ms
    uploadP99: 10000, // ms

    // Error rate
    maxErrorRate: 0.01, // 1%
  },

  // Rate limiting awareness (to avoid hitting rate limits during tests)
  rateLimits: {
    loginAttemptsPerMinute: 5,
    uploadsPerHour: 50,
    reportsPerHour: 10,
  },

  // Think time configuration (simulate realistic user behavior)
  thinkTime: {
    min: 1000, // ms
    max: 5000, // ms

    // Specific scenarios
    afterLogin: 2000,
    betweenImageViews: 3000,
    afterUpload: 5000,
    betweenSocialActions: 2000,
  },
};

// Helper to get full endpoint URL
export function endpoint(path) {
  // Remove leading slash if present to avoid double slashes
  const cleanPath = path.startsWith('/') ? path.slice(1) : path;
  return `${config.baseURL}/${cleanPath}`;
}

// Helper to get random think time
export function getThinkTime(min, max) {
  if (min === undefined || max === undefined) {
    min = config.thinkTime.min;
    max = config.thinkTime.max;
  }
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

// Helper to build query parameters
export function buildQueryParams(params) {
  const query = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null) {
      query.append(key, value.toString());
    }
  });
  const queryString = query.toString();
  return queryString ? `?${queryString}` : '';
}

// Export default configuration
export default config;
