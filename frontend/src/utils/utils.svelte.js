import maplibregl from "maplibre-gl";
import { ZONING_ICON_MAP, CONSTRUCTION_ICON_MAP } from "./icon_map.svelte.js";

import { getFeed, setFeed, setFeedShown } from '../states/feed.svelte.js';

export function _setMarkerBasedOnTags(zoningTag, progressTag) {
  const zone = ZONING_ICON_MAP[zoningTag];
  const construction = CONSTRUCTION_ICON_MAP[progressTag]

  if (!zone) return null;
  if (!construction) return null;

  return [zone, construction]
}

export function addLogoMarker(map, lng, lat, zoningTag, progressTag) {
  const IconComp = _setMarkerBasedOnTags(zoningTag, progressTag);

  if (!IconComp) {
    console.warn("No icon found for:", zoningTag, progressTag);
  }

  const zoneImage = IconComp[0];
  const progressImage = IconComp[1];

  const el = document.createElement("div");
  el.className = "logo-marker";

  el.style.width = "32px";
  el.style.height = "32px";
  el.style.padding = "4px";
  el.style.borderRadius = "50%";
  el.style.display = "flex";
  el.style.justifyContent = "center";
  el.style.alignItems = "center";
  el.style.backgroundColor = "white";
  el.style.backgroundImage = `url(${zoneImage}), url(${progressImage})`;
  el.style.backgroundRepeat = "no-repeat, no-repeat";
  el.style.backgroundSize = "16px 16px, 16px 16px";
  el.style.backgroundPosition = "left center, right center";
  el.style.cursor = "pointer";

  el.addEventListener("click", () => {
    setFeedShown(true);

    const updatedFeed = getFeed().map(item => {
      if (item.longitude === lng && item.latitude === lat) {
        return {
          ...item,
          highlighted: true
        };
      }
      else {
        return {
          ...item,
          highlighted: false
        }
      }
    });

    setFeed(updatedFeed);
  });

  return new maplibregl.Marker({ element: el })
    .setLngLat([lng, lat])
    .addTo(map);
}
