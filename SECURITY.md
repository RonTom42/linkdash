# Security policy

## Supported versions

Only the latest release receives fixes. Please check that a problem still occurs with it before reporting.

## Reporting a vulnerability

Please **do not open a public issue** for security problems. Report them privately with GitHub's private vulnerability reporting: open this repository's **Security** tab and choose **Report a vulnerability**. Include the steps to reproduce, the linkdash version, and what an attacker could achieve.

linkdash is maintained in spare time, so responses are best effort. You can expect an acknowledgement within a week. Once a fix is released, the advisory will be published, crediting you if you like.

## Scope

Examples of what counts as a vulnerability:

- Getting script to run in the generated page through a link name, category, page title or URL (for example by bypassing the escaping or the `javascript:`/`data:` URL check).
- Making linkdash overwrite or damage a file it did not create, or write outside the file it was pointed at.

The page is a static file without a server, so problems that require an attacker to already be able to edit the HTML file directly are out of scope.
