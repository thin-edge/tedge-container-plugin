import { EnvironmentOptions } from '@c8y/devkit/dist/options';
import { author, description, version, license } from './package.json';

const asset = process.env['npm_config_asset'];
const isApp = asset == 'app' ? true : false;

console.log('Building asset:', asset, asset == 'app', isApp);

export default {
  runTime: {
    author,
    description,
    license,
    version,
    name: 'dynamic-mapping',
    contextPath: 'sag-ps-pkg-dynamic-mapping',
    key: 'sag-ps-pkg-dynamic-mapping-key',
    contentSecurityPolicy:
      "base-uri 'none'; default-src 'self' 'unsafe-inline' http: https: ws: wss:; connect-src 'self' http: https: ws: wss:;  script-src 'self' *.bugherd.com *.twitter.com *.twimg.com *.aptrinsic.com 'unsafe-inline' 'unsafe-eval' data:; style-src * 'unsafe-inline' blob:; img-src * data: blob:; font-src * data:; frame-src *; worker-src 'self' blob:;",
    dynamicOptionsUrl: '/apps/public/public-options/options.json',
    remotes: {
      'container-plugin': ['ContainerManagementModule'],
      'Group-plugin': ['GroupManagementModule'],
      'Container-Info-plugin': ['ContainerInfoModule'],
    },
    tabsHorizontal: true,
    noAppSwitcher: false,
    // comment the following properties to create a standalone app
    // comment begin
    // package: 'plugin',
    // isPackage: !isApp,
    package: 'blueprint',
    isPackage: true,
    exports: [
      {
        name: 'Container Info Tab',
        module: 'ContainerInfoModule',
        path: './src/container-info/container-info.module.ts',
        description:
          'Adds a tab to a container service to display all relevant container information.',
      },
      {
        name: 'Container Management Tab',
        module: 'ContainerManagementModule',
        path: './src/container-management/container-management.module.ts',
        description:
          'Adds a tab to a device to monitor the installed containers',
      },
      {
        name: 'Container Group Management Tab',
        module: 'GroupManagementModule',
        path: './src/group-management/group-management.module.ts',
        description:
          'Adds a tab to the device to monitor container groups (aka. docker compose).',
      },
    ],
    // comment end
  },
  buildTime: {
    federation: [
      '@angular/animations',
      '@angular/cdk',
      '@angular/common',
      '@angular/compiler',
      '@angular/core',
      '@angular/forms',
      '@angular/platform-browser',
      '@angular/platform-browser-dynamic',
      '@angular/router',
      '@angular/upgrade',
      '@c8y/client',
      '@c8y/ngx-components',
      'ngx-bootstrap',
      '@ngx-translate/core',
      '@ngx-formly/core',
    ],
  },
} as const satisfies EnvironmentOptions;
