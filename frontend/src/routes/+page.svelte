<script>
	import Header from '../components/header.svelte';
	import debounce from 'lodash/debounce';
	import { onMount, onDestroy } from 'svelte';
	import { setFeed } from '../states/feed.svelte.js';
	import { initMap, getMap } from '../states/map.svelte.js';

	import { api } from '../utils/api.svelte.js';

	let refreshToken = async () => {
		try {
			await api.post('/users/refresh-token');
		} catch (err) {
			console.error('Error refreshing token:', err);
		}
	};

	let retrieveEntries = async (data) => {
		const response = await api.post('/entries/retrieve-entries-within-visible-bounds', data);

		setFeed(response);
	};

	onMount(() => {
		refreshToken();
		initMap('map');

		const mapInstance = getMap();

		const MIN_QUERY_ZOOM = 8;

		const handleMovement = debounce(() => {
			const zoom = mapInstance.getZoom();

			if (zoom < MIN_QUERY_ZOOM) {
				console.log('Zoomed out too far — skipping query');
				setFeed([]);
				return;
			}

			const bounds = mapInstance.getBounds();

			const data = {
				north: bounds.getNorth(),
				south: bounds.getSouth(),
				east: bounds.getEast(),
				west: bounds.getWest()
			};

			retrieveEntries(data);
		}, 1500);

		mapInstance.on('click', (e) => {
			const { lng, lat } = e.lngLat;

			let token = localStorage.getItem('access_token');

			if (token === null) {
				console.log('sorry bucko, not allowed');
				return;
			}

			console.log('Clicked at:', lng, lat);
		});

		mapInstance.on('load', () => {
			mapInstance.resize();

			mapInstance.on('zoomend', () => {
				handleMovement();
			});
			mapInstance.on('moveend', () => {
				console.log('hello?');
				handleMovement();
			});

			handleMovement();
		});

		onDestroy(() => {
			destroyMap();
		});
	});
</script>

<div class="flex h-screen flex-col">
	<Header />
	<div id="map" class="relative min-h-0 min-w-0 flex-1"></div>
</div>
