// SPDX-License-Identifier: GPL-3.0-or-later

import * as vscode from "vscode";
import * as path from "path";
import * as fs from "fs";
import { execSync } from "child_process";
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  Executable,
} from "vscode-languageclient/node";

let client: LanguageClient | undefined;

export function activate(context: vscode.ExtensionContext): void {
  const config = vscode.workspace.getConfiguration("debpack-lsp");

  const binaryPath = resolveBinaryPath(context, config);
  if (!binaryPath) {
    vscode.window.showWarningMessage(
      "debpack-lsp binary not found. Install it with `go install github.com/BAMF0/debpack-lsp@latest` " +
        'or set "debpack-lsp.binaryPath" in settings.',
    );
    return;
  }

  const exec: Executable = {
    command: binaryPath,
    args: [],
  };

  // Optional: enable debug logging via DEBPACK_LSP_LOG env var.
  const logFile = config.get<string>("logFile");
  if (logFile) {
    exec.options = { env: { ...process.env, DEBPACK_LSP_LOG: logFile } };
  }

  const serverOptions: ServerOptions = {
    run: exec,
    debug: exec,
  };

  const clientOptions: LanguageClientOptions = {
    documentSelector: [
      { scheme: "file", pattern: "**/debian/control" },
      { scheme: "file", pattern: "**/debian/changelog" },
      { scheme: "file", pattern: "**/debian/rules" },
      { scheme: "file", pattern: "**/debian/watch" },
      { scheme: "file", pattern: "**/debian/copyright" },
      { scheme: "file", pattern: "**/debian/patches/*" },
      { scheme: "file", pattern: "**/debian/*.install" },
      { scheme: "file", pattern: "**/debian/*.dirs" },
      { scheme: "file", pattern: "**/debian/*.docs" },
      { scheme: "file", pattern: "**/debian/*.links" },
      { scheme: "file", pattern: "**/debian/*.manpages" },
    ],
    synchronize: {
      // Notify the server about config changes so it can re-read sibling files.
      configurationSection: "debpack-lsp",
    },
  };

  client = new LanguageClient(
    "debpack-lsp",
    "Debian Packaging LSP",
    serverOptions,
    clientOptions,
  );

  context.subscriptions.push(client.start());
}

export function deactivate(): Thenable<void> | undefined {
  if (!client) {
    return undefined;
  }
  return client.stop();
}

/**
 * resolveBinaryPath finds the debpack-lsp binary in priority order:
 * 1. The "debpack-lsp.binaryPath" setting (if set)
 * 2. The $PATH (via which/where)
 * 3. A platform-specific binary bundled inside the extension
 */
function resolveBinaryPath(
  context: vscode.ExtensionContext,
  config: vscode.WorkspaceConfiguration,
): string | undefined {
  // 1. Explicit setting.
  const setting = config.get<string>("binaryPath");
  if (setting && fs.existsSync(setting)) {
    return setting;
  }

  // 2. Search $PATH.
  try {
    const cmd =
      process.platform === "win32"
        ? "where debpack-lsp"
        : "which debpack-lsp";
    const found = execSync(cmd, { encoding: "utf8" }).trim();
    if (found && fs.existsSync(found)) {
      return found;
    }
  } catch {
    // not on PATH
  }

  // 3. Bundled binary (cross-compiled, shipped inside the .vsix).
  const arch = process.arch;
  const platformDir = `${process.platform}-${arch}`;
  const binaryName =
    process.platform === "win32" ? "debpack-lsp.exe" : "debpack-lsp";
  const bundled = context.asAbsolutePath(
    path.join("bin", platformDir, binaryName),
  );
  if (fs.existsSync(bundled)) {
    return bundled;
  }

  return undefined;
}
