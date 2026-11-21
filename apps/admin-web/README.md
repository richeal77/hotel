# Admin Web

The admin web uses the existing `/api/v1/admin` contract. It has no direct MySQL connection; the Go API owns database access, staff authentication, authorization, and audit writes.

## Local demo

Start the API and then run `pnpm run dev:admin` from the repository root. Open `http://localhost:5173`. The default proxy target is `http://localhost:8080`; override it at process startup with `ADMIN_API_ORIGIN=http://127.0.0.1:8082`. The proxy accepts only `/__admin_proxy/api/v1/...` and never accepts a target URL from the browser.

## MySQL integration

After the MySQL API has been migrated and staff accounts provisioned as described in `docs/architecture/mysql-phase2.md`, run the admin development server with `APP_ENV=integration ADMIN_API_ORIGIN=http://127.0.0.1:8080 pnpm run dev:admin`. Use the provisioned username and password. The server returns an opaque staff token; the browser keeps it in session storage and clears it on logout or HTTP 401. The API remains the authority for all role checks.

## Static deployment

Run `APP_ENV=production pnpm run build:admin`. Serve `dist/` as static files over HTTPS and route `/api/v1/` on the same origin to the production Go API. The generated `dist/config.js` uses `/api/v1` and disables the development proxy. For a separate HTTPS API origin, set `ADMIN_API_BASE_URL=https://api.example.com/api/v1` during build and configure CORS on that API. Do not put passwords, tokens, or MySQL credentials in build variables.

Production does not expose the Demo console. The frontend shows pages permitted for the role returned by login; the Go API enforces authorization and masks staff responses. Boss and finance users can view the paged audit log. The integration and production logout endpoint revokes the server session.
