# Test Fixtures

These files & directories are referenced in the `*_test.go` files.

## App sources

In `heroku/test-fixtures/`, directories like `app/`, `app-2/`, & `app-broken-build/` can be updated for continued functionality, such as updating the Bundler version in use.

When the app dirs are changed, their associated `*.tgz` archives should be remade with:
```
cd heroku/test-fixtures/
tar -czf app.tgz app/*
tar -czf app-2.tgz app-2/*
tar -czf app-broken-build.tgz app-broken-build/*
```

Then, the **checksums** of these tarballs must be manually computed and updated throughout `heroku/resource_heroku_build_test.go`.
