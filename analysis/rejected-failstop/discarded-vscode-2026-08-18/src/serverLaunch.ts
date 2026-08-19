import * as path from 'path';

export function localServerAddress(port: number): string {
  return `127.0.0.1:${port}`;
}

export function resolvePackageMapPath(serverRoot: string, configuredPath: string | undefined): string | undefined {
  const packageMapPath = configuredPath?.trim();
  if (!packageMapPath) return undefined;
  return path.isAbsolute(packageMapPath)
    ? packageMapPath
    : path.resolve(serverRoot, packageMapPath);
}

// packageMapServeArgs produces literal child-process arguments. It deliberately
// does not invoke a shell, so configured path text is never interpolated.
export function packageMapServeArgs(serverRoot: string, configuredPath: string | undefined): string[] {
  const packageMapPath = resolvePackageMapPath(serverRoot, configuredPath);
  return packageMapPath ? ['--package-map', packageMapPath] : [];
}
