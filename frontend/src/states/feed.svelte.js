import { getMap } from './map.svelte.js';
import maplibregl from 'maplibre-gl';

import { addLogoMarker } from "../utils/utils.svelte.js";

let feed = $state([]);
let feedShown = $state(false)
let activeMarkers = [];

export function getFeed() {
  return feed;
}

export function setFeed(newFeed) {
  feed = newFeed;
  const map = getMap();

  activeMarkers.forEach((marker) => {
    marker.remove();
  });
  activeMarkers = [];

  newFeed.forEach((element) => {
    const zoningName = element.tags.find(t => t.classification === "Zoning").name;
    const progressName = element.tags.find(t => t.classification === "Progress").name;
    const marker = addLogoMarker(
      map,
      element.longitude,
      element.latitude,
      zoningName,
      progressName
    )
    activeMarkers.push(marker);
  });

}

export function getFeedShown() {
  return feedShown;
}

export function setFeedShown(newFeedShown) {
  feedShown = newFeedShown;
}
