# Security Policy

## Reporting a vulnerability

Please report suspected vulnerabilities privately through GitHub Security
Advisories ("Report a vulnerability" on the repository's Security tab) rather
than a public issue. You will get an acknowledgement within a few days.

## What dsmctl stores

- The dsmctl secret is the only secret. On macOS it is kept in the login
  Keychain (service `dsmctl`); with `--no-keychain` or on Linux it is written to
  the config file with mode 0600.
- The token grants full account access, exactly like a logged-in app. Treat it
  as a password. `dsmctl logout` deletes it from the Keychain and the file.
- dsmctl makes requests only to its configured host, or to
  `DSM_URL` if you override it for testing. It sends nothing anywhere else.

## Scope

dsmctl performs only operations an account owner can perform in the eero app. It
does not exploit the router or bypass ownership. APIs may change or restrict access at any time.
