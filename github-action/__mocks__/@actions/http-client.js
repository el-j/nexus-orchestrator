'use strict';

// Manual CJS mock for @actions/http-client v4 (pure-ESM package)
// Jest runs in CommonJS mode and cannot require() the ESM dist directly.

class HttpClient {
  constructor(_userAgent, _handlers, _requestOptions) {
    // intentionally empty — tests override prototype methods via jest.fn()
  }
  postJson(_url, _obj) { return Promise.resolve({ statusCode: 0, result: null, headers: {} }); }
  getJson(_url) { return Promise.resolve({ statusCode: 0, result: null, headers: {} }); }
}

const HttpCodes = {
  OK: 200,
  Accepted: 202,
  MovedPermanently: 301,
  ResourceMoved: 302,
  NotModified: 304,
  BadRequest: 400,
  Unauthorized: 401,
  Forbidden: 403,
  NotFound: 404,
  InternalServerError: 500,
};

module.exports = { HttpClient, HttpCodes };
