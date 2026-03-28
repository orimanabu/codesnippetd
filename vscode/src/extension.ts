import * as vscode from 'vscode';
import * as http from 'http';

export function activate(context: vscode.ExtensionContext): void {
    const disposable = vscode.commands.registerCommand(
        'codesnippetd.sendSelection',
        async () => {
            const editor = vscode.window.activeTextEditor;
            if (!editor) {
                vscode.window.showErrorMessage('codesnippetd: No active editor.');
                return;
            }

            const selection = editor.selection;
            if (selection.isEmpty) {
                vscode.window.showWarningMessage('codesnippetd: No text selected.');
                return;
            }

            const text = editor.document.getText(selection);
            const config = vscode.workspace.getConfiguration('codesnippetd');
            const host = config.get<string>('host', 'localhost');
            const port = config.get<number>('port', 8999);

            try {
                await postToPipe(host, port, text);
                vscode.window.showInformationMessage(
                    `codesnippetd: Selection sent to http://${host}:${port}/pipe`
                );
            } catch (err) {
                vscode.window.showErrorMessage(`codesnippetd: POST failed: ${err}`);
            }
        }
    );

    context.subscriptions.push(disposable);
}

function postToPipe(host: string, port: number, body: string): Promise<void> {
    return new Promise((resolve, reject) => {
        const data = Buffer.from(body, 'utf-8');
        const req = http.request(
            {
                hostname: host,
                port,
                path: '/pipe',
                method: 'POST',
                headers: {
                    'Content-Type': 'application/octet-stream',
                    'Content-Length': data.length,
                },
            },
            (res) => {
                // Consume the response body to free the socket.
                res.resume();
                if (res.statusCode !== undefined && res.statusCode >= 400) {
                    reject(new Error(`HTTP ${res.statusCode}`));
                } else {
                    resolve();
                }
            }
        );
        req.on('error', reject);
        req.write(data);
        req.end();
    });
}

export function deactivate(): void {}
