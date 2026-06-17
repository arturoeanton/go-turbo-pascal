import * as vscode from "vscode";
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind,
} from "vscode-languageclient/node";

let client: LanguageClient;

export function activate(context: vscode.ExtensionContext) {
  const config = vscode.workspace.getConfiguration("bpgoPascal");
  const command = config.get<string>("serverPath", "pls");

  const serverOptions: ServerOptions = {
    run: { command, transport: TransportKind.stdio },
    debug: { command, transport: TransportKind.stdio },
  };

  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: "file", language: "pascal" }],
  };

  client = new LanguageClient(
    "bpgoPascal",
    "BPGo Pascal Language Server",
    serverOptions,
    clientOptions
  );

  client.start();

  // Debug adapter: spawn the pdap executable for the "bpgo-pascal" debug type.
  const adapter = config.get<string>("debugAdapterPath", "pdap");
  context.subscriptions.push(
    vscode.debug.registerDebugAdapterDescriptorFactory("bpgo-pascal", {
      createDebugAdapterDescriptor() {
        return new vscode.DebugAdapterExecutable(adapter, []);
      },
    })
  );
}

export function deactivate(): Thenable<void> | undefined {
  return client ? client.stop() : undefined;
}
