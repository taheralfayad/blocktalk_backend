<script lang="ts">
  import Entry from "../components/entry.svelte";
  import rightArrow from "$lib/assets/right.svg";
  import {
    getDistance,
    getLocationSearchValue,
  } from "../states/searchBarState.svelte.js";
  import { fly } from "svelte/transition";

  import { api } from "../utils/api.svelte.js";

  import {
    getFeedShown,
    setFeedShown,
    getFeed,
    setFeed,
  } from "../states/feed.svelte.js";

  let feedShown = $derived(getFeedShown());
  let feed: any[] = $state([]);

  const closeMenu = () => {
    const clearedFeed = getFeed().map((item) => ({
      ...item,
      highlighted: false,
    }));

    setFeed(clearedFeed);
    setFeedShown(false);
  };
  let isLoggedIn = $state(false);

  const getIsLoggedIn = async () => {
    try {
      await api.get("/users/me");
      isLoggedIn = true;
    } catch (error) {
      if (error.status === 401) {
        return false;
      }

      return false;
    }
  };

  $effect(() => {
    feed = getFeed();
    getIsLoggedIn();
  });
</script>

{#if feedShown}
  <div class="fixed inset-0 z-50 flex" transition:fly|global>
    <div class="flex-1 bg-black opacity-25" onclick={closeMenu}></div>
    <div
      class="relative flex h-full w-full transform flex-col bg-white shadow-lg sm:w-2/5"
    >
      <button class="hover:cursor-pointer" onclick={closeMenu}>
        <img class="max-h-8 max-w-8" src={rightArrow} alt="Close" />
      </button>
      <div class="flex flex-1 flex-col overflow-y-auto p-4">
        {#if feed.length > 0}
          {#each feed as item}
            <Entry
              title={item.title}
              address={item.address}
              content={item.content}
              zoningTag={item.tags.find(
                (tag) => tag.classification === "Zoning",
              ).name}
              progressTag={item.tags.find(
                (tag) => tag.classification === "Progress",
              ).name}
              highlighted={item.highlighted ?? false}
            />
          {/each}
        {:else}
          <p>No entries found for this area :(</p>
          <p>
            You can add some by joining the BlockTalk community as a
            contributor!
          </p>
        {/if}
      </div>

      <div class="border-t border-gray-200">
        <a
          href="/about"
          class="block w-full px-4 py-3 text-left transition-colors hover:bg-gray-50"
        >
          About
        </a>

        {#if isLoggedIn}
          <a
            href="/create-entry"
            class="block w-full px-4 py-3 text-left transition-colors hover:bg-gray-50"
          >
            Participate
          </a>
        {:else}
          <a
            href="/login"
            class="block w-full px-4 py-3 text-left font-semibold transition-colors hover:bg-gray-50"
          >
            Login
          </a>
        {/if}
      </div>
    </div>
  </div>
{/if}
