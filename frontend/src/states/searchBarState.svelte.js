let distance = $state(10); // start off with 10 miles as the default value
let locationSearchValue = $state("");

export function getDistance() {
  return distance;
}

export function setDistance(newDistance) {
  distance = newDistance;
}

export function getLocationSearchValue() {
  return locationSearchValue;
}

export function setLocationSearchValue(newLocationSearchValue) {
  locationSearchValue = newLocationSearchValue;
}
