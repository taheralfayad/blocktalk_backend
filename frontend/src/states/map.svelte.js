import maplibregl from 'maplibre-gl';

let map = null;

export function initMap(container) {
  map = new maplibregl.Map({
    container,
    style: 'https://tiles.openfreemap.org/styles/liberty',
    center: [-81.3792, 28.5383],
    zoom: 10,
    attributionControl: false
  });

  map.addControl(new maplibregl.NavigationControl(), 'top-right');

  return map;
}

export function getMap() {
  return map;
}

export function destroyMap() {
  if (map) {
    map.remove();
    map = null;
  }
}
