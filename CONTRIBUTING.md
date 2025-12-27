<!--
SPDX-FileCopyrightText: 2024 Ville Eurométropole Strasbourg

SPDX-License-Identifier: MIT
-->

# Contributing to Gristle

Thank you for your interest in contributing to **Gristle**! We welcome all contributions, whether they involve reporting bugs, suggesting new features, improving documentation, or submitting code.

## Before You Begin

1. **Read the documentation**: Familiarize yourself with the purpose and functionality of the project by reviewing the [README.md](./README.md).
2. **Check existing issues**: Before creating a new issue or suggesting a feature, browse through the open issues to avoid duplicates.
3. **Follow the code of conduct**: Please ensure you adhere to our [Code of Conduct](./CODE_OF_CONDUCT.md).

## How to Contribute?

### 1. Reporting a Bug

1. Go to the Issues tab.
2. Create a new issue using the "Bug Report" template.
3. Provide as much detail as possible:
   - Steps to reproduce the problem
   - Expected vs. actual results
   - Environment (OS, gristle version, etc.)
   - Relevant logs or error messages

### 2. Suggesting an Improvement or New Feature

1. Check if a similar suggestion already exists.
2. Create a new issue using the "Feature Request" template.
3. Clearly describe the proposed improvement or feature and explain its benefits.

### 3. Submitting Code

#### Step 1: Fork the Project

- Create a fork of the main repository on your GitHub account.
- Clone your fork locally:
  ```bash
  git clone https://github.com/<your-username>/gristle.git
  cd gristle
  ```

#### Step 2: Create a Branch

Work in a branch specific to your contribution:

```bash
git checkout -b my-contribution
```

#### Step 3: Make Your Changes

Ensure your code adheres to the existing conventions.
If adding a new feature, include appropriate tests.

#### Step 4: Test Your Code

Run tests to verify that your changes do not introduce any issues:

```bash
go test ./...
```

#### Step 5: Submit a Pull Request

Push your branch to your fork:

```bash
git push origin my-contribution
```

Open a Pull Request (PR) in the main repository.
Provide a clear description of your changes in the PR form.

### 4. Improving Documentation

Documentation is crucial. You can contribute by fixing errors, adding details, or translating content.
Submit your documentation improvements through a Pull Request as explained above.

## Best Practices

Follow coding standards: Ensure your code is clean, well-documented, and consistent with the project's style.
Write clear commit messages: Summarize what you've done concisely and accurately.
Test your changes: Make sure your contribution doesn't introduce regressions.

## Need Help?

If you have any questions, feel free to open an issue or reach out to the maintainers. We're here to help!

Thank you for contributing to Gristle!
