import os
import re

for root, _, files in os.walk('.'):
    for f in files:
        if not f.endswith('test.go') and not f.endswith('main.go'):
            continue
        path = os.path.join(root, f)
        with open(path, 'r') as file:
            content = file.read()
        
        # Replace memRepo
        if 'func (m *memRepo) GetTasksBySessionID' in content and 'GetStaleProcessing' not in content:
            content = re.sub(
                r'(func \(m \*memRepo\) GetTasksBySessionID\(sessionID string\) \(\[\]domain.Task, error\) \{.*?^\})',
                r'\1\n\nfunc (m *memRepo) GetStaleProcessing(ctx context.Context, threshold time.Duration) ([]domain.Task, error) { return nil, nil }',
                content,
                flags=re.MULTILINE | re.DOTALL
            )
            with open(path, 'w') as file:
                file.write(content)
        
        # Replace optRepo
        elif 'func (o *optRepo) GetTasksBySessionID' in content and 'GetStaleProcessing' not in content:
            content = re.sub(
                r'(func \(o \*optRepo\) GetTasksBySessionID\(sessionID string\) \(\[\]domain.Task, error\) \{.*?^\})',
                r'\1\n\nfunc (o *optRepo) GetStaleProcessing(ctx context.Context, threshold time.Duration) ([]domain.Task, error) { return nil, nil }',
                content,
                flags=re.MULTILINE | re.DOTALL
            )
            with open(path, 'w') as file:
                file.write(content)

