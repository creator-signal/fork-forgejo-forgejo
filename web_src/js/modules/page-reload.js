let reloadFn = () => window.location.reload();

export function requestPageReload() {
  reloadFn();
}

export function setPageReloadFn(fn) {
  const prev = reloadFn;
  reloadFn = fn;
  return () => { reloadFn = prev };
}
