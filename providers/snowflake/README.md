# Snowflake Provider

```shell
cnquery shell snowflake
```

Required arguments:

- `--account` - The Snowflake Data Cloud account name.
- `--region` - The Snowflake Data Cloud region.
- `--user` - The Snowflake Data Cloud username.
- `--role` - The Snowflake Data Cloud role.

> The easiest way to get the account name and region is to look at the URL when you log in to the Snowflake web interface. When clicking on the account icon you can copy the account URL that included the account name and region.

**Password Authentication**

Arguments:

- `--password` - The Snowflake Data Cloud password.
- `--ask-pass` - Prompt for the Snowflake Data Cloud password.

```shell
shell snowflake --account zi12345 --region us-central1.gcp --user CHRIS  --role ACCOUNTADMIN --ask-pass
```

> To create a username and password, use [Snowsight](https://docs.snowflake.com/en/user-guide/admin-user-management#using-snowsight) or using [SQL](https://docs.snowflake.com/en/user-guide/admin-user-management#using-sql).

**Certificate Authentication**

Arguments:

- `--private-key` - The path to the private key file.

```shell
shell snowflake --account zi12345 --region us-central1.gcp --user CHRIS  --role ACCOUNTADMIN --private-key ~/.ssh/id_rsa
```

> You need to generate a RSA key pair and assign the public key to your user via [Snowsight](https://docs.snowflake.com/en/user-guide/key-pair-auth).

## Examples

**Retrieve all users**

```shell
cnquery> snowflake.account.users
snowflake.account.users: [
  0: snowflake.user name="CHRIS"
  1: snowflake.user name="DATAUSER"
  2: snowflake.user name="SNOWFLAKE"
]
```

**Retrieve all users that have no MFA**

```shell
cnquery> snowflake.account.users.where(extAuthnDuo == false)
snowflake.account.users.where: [
  0: snowflake.user name="CHRIS"
  1: snowflake.user name="DATAUSER"
  2: snowflake.user name="SNOWFLAKE"
]
```

**Retrieve all users that have password authentication**

```shell
cnquery> snowflake.account.users.where(hasPassword)
snowflake.account.users.where: [
  0: snowflake.user name="CHRIS"
  1: snowflake.user name="DATAUSER"
  2: snowflake.user name="SNOWFLAKE"
]

```

**Retrieve all users that have certificate authentication**

```shell
cnquery> snowflake.account.users.where(hasRsaPublicKey)
snowflake.account.users.where: [
  0: snowflake.user name="CHRIS"
]
```

**Retrieve users that have not logged in for 30 days**

```shell
cnquery> snowflake.account.users.where(time.now - lastSuccessLogin > time.day * 30) { lastSuccessLogin }
snowflake.account.users.where: [
  0: {
    lastSuccessLogin: 366 days
  }
]
```

**Check that SCIM is enabled**

```shell
cnquery> snowflake.account.securityIntegrations.where(type == /SCIM/).any(enabled == true)
[failed] [].any()
  actual:   []
```

**Check the retention time is greater 90 days**

```shell
cnquery> snowflake.account.parameters.one(key == "DATA_RETENTION_TIME_IN_DAYS" && value >= 90)
```

**Retrieve all databases**

```shell
cnquery> snowflake.account.databases
snowflake.account.databases: [
  0: snowflake.database name="CNQUERY"
  1: snowflake.database name="SNOWFLAKE"
  2: snowflake.database name="SNOWFLAKE_SAMPLE_DATA"
]
```
