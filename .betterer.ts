import { regexp } from '@betterer/regexp';
import { BettererFileTest } from '@betterer/betterer';
import { ESLint, Linter } from 'eslint';
import { existsSync } from 'fs';
import { exec } from 'child_process';
import path from 'path';
import glob from 'glob';

export default {
  'no enzyme tests': () => regexp(/from 'enzyme'/g).include('**/*.test.*'),
  'better eslint': () => countEslintErrors().include('**/*.{ts,tsx}'),
  'no undocumented stories': () => countUndocumentedStories().include('**/*.story.tsx'),
};

function countUndocumentedStories() {
  return new BettererFileTest(async (filePaths, fileTestResult) => {
    filePaths.forEach((filePath) => {
      if (!existsSync(filePath.replace(/\.story.tsx$/, '.mdx'))) {
        // In this case the file contents don't matter:
        const file = fileTestResult.addFile(filePath, '');
        // Add the issue to the first character of the file:
        file.addIssue(0, 0, 'No undocumented stories are allowed, please add an .mdx file with some documentation');
      }
    });
  });
}

async function findEslintConfigFiles(): Promise<string[]> {
  return new Promise((resolve, reject) => {
    glob('**/.eslintrc', (err, files) => {
      if (err) {
        reject(err);
      }
      resolve(files);
    });
  });
}

function countEslintErrors() {
  return new BettererFileTest(async (filePaths, fileTestResult, resolver) => {
    const { baseDirectory } = resolver;
    const cli = new ESLint({ cwd: baseDirectory });

    const eslintConfigFiles = await findEslintConfigFiles();
    const eslintConfigMainPaths = eslintConfigFiles.map((file) => path.resolve(path.dirname(file)));

    const baseRules: Partial<Linter.RulesRecord> = {
      '@typescript-eslint/no-explicit-any': 'error',
      '@grafana/no-aria-label-selectors': 'error',
    };

    const nonTestFilesRules: Partial<Linter.RulesRecord> = {
      ...baseRules,
      '@typescript-eslint/consistent-type-assertions': ['error', { assertionStyle: 'never' }],
    };

    // group files by eslint config file
    // this will create two file groups for each eslint config file
    // one for test files and one for non-test files
    const fileGroups: Record<string, string[]> = {};

    for (const filePath of filePaths) {
      let configPath = eslintConfigMainPaths.find((configPath) => filePath.startsWith(configPath)) ?? '';
      const isTestFile =
        filePath.endsWith('.test.tsx') ||
        filePath.endsWith('.test.ts') ||
        filePath.includes('__mocks__') ||
        filePath.includes('public/test/');

      if (isTestFile) {
        configPath += '-test';
      }
      if (!fileGroups[configPath]) {
        fileGroups[configPath] = [];
      }
      fileGroups[configPath].push(filePath);
    }

    for (const configPath of Object.keys(fileGroups)) {
      const rules = configPath.endsWith('-test') ? baseRules : nonTestFilesRules;
      // this is by far the slowest part of this code. It takes eslint about 2 seconds just to find the config
      const linterOptions = (await cli.calculateConfigForFile(fileGroups[configPath][0])) as Linter.Config;
      const runner = new ESLint({
        baseConfig: {
          ...linterOptions,
          rules: rules,
        },
        useEslintrc: false,
        cwd: baseDirectory,
      });
      const lintResults = await runner.lintFiles(fileGroups[configPath]);
      lintResults
        .filter((lintResult) => lintResult.source)
        .forEach((lintResult) => {
          const { messages } = lintResult;
          const filePath = lintResult.filePath;
          const file = fileTestResult.addFile(filePath, '');
          messages.forEach((message, index) => {
            file.addIssue(0, 0, message.message, `${index}`);
          });
        });
    }
  });
}
