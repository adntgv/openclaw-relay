import { mkdir, cp } from 'node:fs/promises';

await mkdir('dist', { recursive: true });
await cp('src', 'dist/src', { recursive: true });
await cp('protocol', 'dist/protocol', { recursive: true });
console.log('Release assets prepared in dist/');
