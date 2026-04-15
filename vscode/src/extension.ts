import * as vscode from 'vscode';
import * as http from 'http';
import * as fs from 'fs';
import * as path from 'path';

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
            const filePath = gitRelativePath(editor.document.fileName);
            const startLine = selection.start.line + 1;
            const endLine = selection.end.line + 1;
            const config = vscode.workspace.getConfiguration('codesnippetd');
            const host = config.get<string>('host', 'localhost');
            const port = config.get<number>('port', 8999);

            const payload = JSON.stringify({
                name: '',
                path: filePath,
                start: startLine,
                end: endLine,
                code: text,
            });

            try {
                await postToPipe(host, port, payload);
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
                    'Content-Type': 'application/json',
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

function gitRelativePath(fullPath: string): string {
    let dir = path.dirname(fullPath);
    while (true) {
        if (fs.existsSync(path.join(dir, '.git')) && fs.statSync(path.join(dir, '.git')).isDirectory()) {
            return path.relative(dir, fullPath);
        }
        const parent = path.dirname(dir);
        if (parent === dir) {
            return fullPath;
        }
        dir = parent;
    }
}

export function deactivate(): void {}
