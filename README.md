# Selebrow Project

Selebrow is an open-source tool that simplifies running UI tests with WebDriver-based frameworks like
[Selenium](https://www.selenium.dev/), [Selenide](https://selenide.org/), [WebdriverIO](https://webdriver.io/) (wdio), and [Playwright](https://playwright.dev/).


It can serve as a drop-in replacement for the discontinued [Selenoid](https://github.com/aerokube/selenoid) project allowing running browsers in Docker containers or
Kubernetes pods.

Selebrow allows running tests in multiple browsers concurrently using prebuilt [browser images](https://selebrow.dev/docs/concepts/images/),
ensuring reliable results both locally and in CI.

## Key features

* [Kubernetes backend](https://selebrow.dev/docs/concepts/backend/#kubernetes) support
* Ability to run as [GitLab CI service](https://selebrow.dev/docs/start/gitlab-ci/)
* Support for running [Playwright tests](https://selebrow.dev/docs/usage/playwright/)
* [Browser pooling](https://selebrow.dev/docs/concepts/pooling/) for faster tests startup
* [UI](https://selebrow.dev/docs/concepts/ui/) integrated directly into binary, no separate components required

## Resources

* [Documentation](https://selebrow.dev/docs/intro/)
* [Browser images](https://github.com/selebrow/images)


