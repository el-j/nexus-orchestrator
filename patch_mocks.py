import os
import re

for root, _, files in os.walk('.'):
    for f in files:
        path = os.path.join(root, f)
        if not path.endswith('.go'):
            continue
        with open(path, 'r') as file:
            content = file.read()
        
        # Replace mockOrchestrator
        if 'func (m *mockOrchestrator) HeartbeatAISession' in content and 'HeartbeatTask' not in content:
            content = re.sub(
                r'(func \(m \*mockOrchestrator\) HeartbeatAISession\(.*?\) error \{.*?\})',
                r'\1\nfunc (m *mockOrchestrator) HeartbeatTask(ctx context.Context, taskID, sessionID string) error { return nil }',
                content,
                flags=re.MULTILINE | re.DOTALL
            )
            with open(path, 'w') as file:
                file.write(content)
        
        # Replace mockOrch
        elif 'func (m *mockOrch) HeartbeatAISession' in content and 'HeartbeatTask' not in content:
            content = re.sub(
                r'(func \(m \*mockOrch\) HeartbeatAISession\(.*?\) error \{.*?\})',
                r'\1\nfunc (m *mockOrch) HeartbeatTask(ctx context.Context, taskID, sessionID string) error { return nil }',
                content,
                flags=re.MULTILINE | re.DOTALL
            )
            with open(path, 'w') as file:
                file.write(content)

