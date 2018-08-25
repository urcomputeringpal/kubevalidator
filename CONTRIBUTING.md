## Contributing

[fork]: https://github.com/urcomputeringpal/kubevalidator/fork
[pr]: https://github.com/urcomputeringpal/kubevalidator/compare
[style]: https://github.com/golang/go/wiki/Style
[code-of-conduct]: CODE_OF_CONDUCT.md

Hi there! We're thrilled that you'd like to contribute to this project. Your help is essential for keeping it great.

Please note that this project is released with a [Contributor Code of Conduct][code-of-conduct]. By participating in this project you agree to abide by its terms.

## Submitting a pull request

0. Optional: Open an issue describing your intended changes if you'd like! I'm happy to discuss any changes you'd like to see or provide feedback on a proposed design.
0. [Fork][fork] and clone the repository.
0. Run the tests with `go test ./validator` (or by installing the [Google Cloud Build GitHub App on your fork](https://github.com/apps/google-container-builder)). Verify they work, and open an issue if not.
0. Create a new branch: `git checkout -b my-branch-name`
0. Make your changes.
0. Add tests to confirm your changes. Verify that they pass.
0. Push to your fork and [submit a pull request][pr] describing your changes and the motivation behind them.
0. Pat your self on the back and wait for your pull request to be reviewed and merged. Have patience if it takes a minute.

Here are a few things you can do that will increase the likelihood of your pull request being accepted:

- Follow the [style guide][style] and format your code using `goimports`, `gofmt`, or something similar.
- Keep your change as focused as possible. If there are multiple changes you would like to make that are not dependent upon each other, consider submitting them as separate pull requests.
- Write a [good commit message](http://tbaggery.com/2008/04/19/a-note-about-git-commit-messages.html).

Work in Progress pull requests are also welcome to get feedback early on, or if there is something blocked you. Please prefix the title of your PR with `[WIP]` if your changes are still in progress!

## Resources

- [How to Contribute to Open Source](https://opensource.guide/how-to-contribute/)
- [Using Pull Requests](https://help.github.com/articles/about-pull-requests/)
- [GitHub Help](https://help.github.com)
