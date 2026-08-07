// Seed fixtures for the mongo provider integration test.
// Load with: mongosh -u admin -p <pw> --authenticationDatabase admin seed.js

// A custom role in appdb with a narrow privilege.
db.getSiblingDB("appdb").createRole({
  role: "appReadMetrics",
  privileges: [
    { resource: { db: "appdb", collection: "metrics" }, actions: ["find"] },
  ],
  roles: [{ role: "read", db: "appdb" }],
});

// A low-privilege application user granted the custom role.
db.getSiblingDB("appdb").createUser({
  user: "appuser",
  pwd: "Str0ng!Passw0rd2026",
  roles: [{ role: "appReadMetrics", db: "appdb" }],
});

// A user holding a high-privilege built-in role (superuser-review fixture).
db.getSiblingDB("admin").createUser({
  user: "opsadmin",
  pwd: "Str0ng!Passw0rd2026",
  roles: [{ role: "readWriteAnyDatabase", db: "admin" }],
});

print("seed complete");
