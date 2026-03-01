const BASE_URL = 'http://localhost:8080';

/**
 * @param {string} route - API route (e.g. "/users", "/login")
 * @param {Object} options
 * @param {"GET"|"POST"|"PUT"|"PATCH"|"DELETE"} [options.method="GET"]
 * @param {Object|null} [options.body=null] - JSON body for non-GET requests
 * @param {Object} [options.headers={}] - Additional headers
 */
export async function apiRequest(
  route,
  {
    method = 'GET',
    body = null,
    headers = {},
  } = {}
) {
  const config = {
    method,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...headers,
    },
  };

  if (body && method !== 'GET') {
    config.body = JSON.stringify(body);
  }

  const response = await fetch(`${BASE_URL}${route}`, config);

  let data;
  const contentType = response.headers.get('content-type');

  if (contentType && contentType.includes('application/json')) {
    data = await response.json();
  } else {
    data = await response.text();
  }

  if (!response.ok) {
    throw {
      status: response.status,
      data,
    };
  }

  return data;
}

/**
 * Convenience helpers
 */
export const api = {
  get: (route, headers = {}) =>
    apiRequest(route, { method: 'GET', headers }),

  post: (route, body, headers = {}) =>
    apiRequest(route, { method: 'POST', body, headers }),

  put: (route, body, headers = {}) =>
    apiRequest(route, { method: 'PUT', body, headers })
};


