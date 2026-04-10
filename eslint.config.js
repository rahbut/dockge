import eslint from "@eslint/js";
import tseslint from "typescript-eslint";
import pluginVue from "eslint-plugin-vue";
import jsdoc from "eslint-plugin-jsdoc";
import stylistic from "@stylistic/eslint-plugin";
import globals from "globals";

export default [
    // Global ignores
    {
        ignores: [
            "dist/",
            "dist-frontend/",
            "frontend-dist/",
            "node_modules/",
            "extra/",
            "frontend/components.d.ts",
            ".eslintrc.cjs",
        ],
    },

    // Base configs
    eslint.configs.recommended,
    ...tseslint.configs.recommended,
    ...pluginVue.configs["flat/recommended"],

    // Vue files need TypeScript parser for <script lang="ts">
    {
        files: ["**/*.vue"],
        languageOptions: {
            parserOptions: {
                parser: tseslint.parser,
            },
        },
    },

    // Custom rules for all TS/Vue files
    {
        files: ["**/*.{ts,vue}"],
        languageOptions: {
            globals: {
                ...globals.browser,
                ...globals.node,
            },
        },
        plugins: {
            jsdoc,
            "@stylistic": stylistic,
        },
        rules: {
            // Core rules
            "yoda": "error",
            "camelcase": [ "warn", {
                "properties": "never",
                "ignoreImports": true
            }],
            "no-unused-vars": [ "warn", {
                "args": "none"
            }],
            "curly": "error",
            "no-var": "error",
            "no-constant-condition": [ "error", {
                "checkLoops": false,
            }],
            "no-extra-boolean-cast": "off",
            "no-empty": [ "error", {
                "allowEmptyCatch": true
            }],
            "no-control-regex": "off",
            "one-var": [ "error", "never" ],
            "prefer-const": "off",

            // Stylistic rules (moved from deprecated eslint core)
            "@stylistic/linebreak-style": [ "error", "unix" ],
            "@stylistic/indent": [
                "error",
                4,
                {
                    ignoredNodes: [ "TemplateLiteral" ],
                    SwitchCase: 1,
                },
            ],
            "@stylistic/quotes": [ "error", "double" ],
            "@stylistic/semi": "error",
            "@stylistic/no-multi-spaces": [ "error", {
                ignoreEOLComments: true,
            }],
            "@stylistic/array-bracket-spacing": [ "warn", "always", {
                "singleValue": true,
                "objectsInArrays": false,
                "arraysInArrays": false
            }],
            "@stylistic/space-before-function-paren": [ "error", {
                "anonymous": "always",
                "named": "never",
                "asyncArrow": "always"
            }],
            "@stylistic/object-curly-spacing": [ "error", "always" ],
            "@stylistic/object-curly-newline": "off",
            "@stylistic/object-property-newline": [ "error", {
                allowAllPropertiesOnSameLine: true,
            }],
            "@stylistic/comma-spacing": "error",
            "@stylistic/brace-style": "error",
            "@stylistic/key-spacing": "warn",
            "@stylistic/keyword-spacing": "warn",
            "@stylistic/space-infix-ops": "error",
            "@stylistic/arrow-spacing": "warn",
            "@stylistic/no-trailing-spaces": "error",
            "@stylistic/space-before-blocks": "warn",
            "@stylistic/no-multiple-empty-lines": [ "warn", {
                "max": 1,
                "maxBOF": 0,
            }],
            "@stylistic/lines-between-class-members": [ "warn", "always", {
                exceptAfterSingleLine: true,
            }],
            "@stylistic/array-bracket-newline": [ "error", "consistent" ],
            "@stylistic/eol-last": [ "error", "always" ],
            "@stylistic/comma-dangle": [ "warn", "only-multiline" ],
            "@stylistic/max-statements-per-line": [ "error", { "max": 1 }],

            // Vue rules
            "vue/html-indent": [ "error", 4 ], // default: 2
            "vue/max-attributes-per-line": "off",
            "vue/singleline-html-element-content-newline": "off",
            "vue/html-self-closing": "off",
            "vue/require-component-is": "off",      // not allow is="style" https://github.com/vuejs/eslint-plugin-vue/issues/462#issuecomment-430234675
            "vue/attribute-hyphenation": "off",     // This change noNL to "no-n-l" unexpectedly
            "vue/multi-word-component-names": "off",

            // TypeScript rules
            "@typescript-eslint/ban-ts-comment": "off",
            "@typescript-eslint/no-unused-vars": [ "warn", {
                "args": "none"
            }],
        },
    },
];
