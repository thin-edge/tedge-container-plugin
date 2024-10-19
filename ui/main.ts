import '../../cumulocity-dynamic-mapper/dynamic-mapping-ui/src/i18n';
import { applyOptions, loadOptions } from '@c8y/bootstrap';

const barHolder: HTMLElement = document.querySelector('body > .init-load');
export const removeProgress = () =>
  barHolder && barHolder.parentNode.removeChild(barHolder);

applicationSetup();

async function applicationSetup() {
  const options = await applyOptions({
    ...(await loadOptions()),
  });

  const mod = await import(
    '../../cumulocity-dynamic-mapper/dynamic-mapping-ui/src/bootstrap'
  );
  const bootstrapApp =
    mod.bootstrap || (window as any).bootstrap || (() => null);

  return Promise.resolve(bootstrapApp(options)).then(removeProgress);
}
