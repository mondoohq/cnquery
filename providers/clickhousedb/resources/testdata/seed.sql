-- Copyright Mondoo, Inc. 2024, 2026
-- SPDX-License-Identifier: BUSL-1.1
--
-- Seeds security-relevant fixtures the integration test asserts on. Run against
-- a server with SQL-driven access management enabled, as an account that can
-- manage access (for example the default user), for example:
--
--   clickhouse-client --multiquery < seed.sql
--
-- No credentials are set here on purpose (nothing to leak); the fixtures the
-- test needs are a password-less account, a host restriction, and a broad grant.

-- A password-less user: can log in without a credential (a finding).
CREATE USER IF NOT EXISTS weakuser IDENTIFIED WITH no_password;

-- A role granted broad SELECT on everything, and a user that holds it. The user
-- is restricted to a host range so the test can assert it is not any-host.
CREATE ROLE IF NOT EXISTS analyst;
GRANT SELECT ON *.* TO analyst;
CREATE USER IF NOT EXISTS appuser IDENTIFIED WITH no_password HOST IP '10.0.0.0/8';
GRANT analyst TO appuser;
