// Split out from client.js so api/auth.js can throw the same error shape
// without an import cycle (client.js needs auth.js for the 401 retry path).
export class ApiError extends Error {
  constructor(status, code, message) {
    super(message);
    this.status = status;
    this.code = code;
  }
}
