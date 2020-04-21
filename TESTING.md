# Testing

## Provider Tests
In order to test the provider, you can simply run `make test`.

```bash
$ make test
```

## Acceptance Tests

**Acceptance tests create real resources and cost money to run. You are responsible for any costs incurred!**

You can run the complete suite of Heroku acceptance tests by doing the following:

```bash
$ make testacc TEST="./heroku/" 2>&1 | tee test.log
```

To run a single acceptance test in isolation replace the last line above with:

```bash
$ make testacc TEST="./heroku/" TESTARGS='-run=TestAccHerokuSpace_Basic'
```

A set of tests can be selected by passing `TESTARGS` a substring. For example, to run all Heroku Private Space tests:

```bash
$ make testacc TEST="./heroku/" TESTARGS='-run=HerokuSpace'
```

### Test Parameters

The following parameters are available for running the test. The absence of some of the non-required parameters will cause certain tests to be skipped.

* **HEROKU_API_KEY**(`string`) **Required** The api key of the user running the test.
* **HEROKU_EMAIL**(`string`) **Required** The email of the user running the test.
* **HEROKU_ORGANIZATION**(`string`) **Required** The Heroku Team in which tests will be run.
* **HEROKU_TEST_USER**(`string`) The name of an existing user belonging to the organization, that will be used for various test cases.
* **HEROKU_NON_ADMIN_TEST_USER**(`string`) The name of an existing non-admin user belonging to the organization, that will be used for various test cases.
* **HEROKU_SLUG_ID**(`string`) The ID of an existing slug built in the Common Runtime (otherwise "Slug not compatible with space" errors will be thrown)
* **HEROKU_SPACES_ORGANIZATION**(`string`) The Heroku Enterprise Team for which Heroku Private Space tests will be run under.
* **HEROKU_USER_ID**(`string`) The UUID of an existing Heroku user.
* **TF_LOG**(`DEBUG|TRACE`) Enables more detailed logging of tests, including http request/responses. 

For example:

```bash
export HEROKU_EMAIL=...
export HEROKU_API_KEY=...
export HEROKU_ORGANIZATION='my-heroku-org'
export HEROKU_SPACES_ORGANIZATION='my-heroku-org'
export HEROKU_TEST_USER='admin-user@myco.com'
export HEROKU_NON_ADMIN_TEST_USER='non-admin-user@myco.com'
$ make testacc TEST="./heroku/" 2>&1 | tee test.log
```
