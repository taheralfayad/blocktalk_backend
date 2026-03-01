<script>
  import Retvrn from "../../components/retvrn.svelte";
  import Input from "../../components/input.svelte";
  import SelectInput from "../../components/select.svelte";
  import DropdownTextfield from "../../components/dropdownTextfield.svelte";
  import Button from "../../components/button.svelte";
  import Tags from "../../globals/tags.json";

  import { api } from "../../utils/api.svelte";

  let title = $state("");
  let location = $state("");
  let latitude = $state("");
  let longitude = $state("");
  let zoningTag = $state("");
  let progressTag = $state("");
  let description = $state("");
  let error = $state("");
  let success = $state(false);
  let submitting = $state(false);
  let suggestions = $state([]);

  const suggestionsHidden = $derived(suggestions.length === 0);

  let handleLocationAutosuggestion = async (event) => {
    if (location.length > 3) {
      const data = await api.get(
        `/entries/autocomplete-address?query=${location}`,
      );

      suggestions = data;
    } else {
      suggestions = [];
    }
  };

  let handleSelectSuggestion = (suggestionName) => {
    const suggestionObject = suggestions.find(
      (suggestion) => suggestion.address == suggestionName,
    );

    location = suggestionObject.address;
    longitude = suggestionObject.lon;
    latitude = suggestionObject.lat;
    suggestions = [];
  };

  let handleGetCurrentLocation = async () => {
    if (!navigator.geolocation) {
      error = "Geolocation is not supported by your browser";
      return;
    }

    try {
      const position = await new Promise((resolve, reject) => {
        navigator.geolocation.getCurrentPosition(resolve, reject);
      });

      const { latitude, longitude } = position.coords;

      const data = await api.get(
        `https://nominatim.openstreetmap.org/reverse?format=json&lat=${latitude}&lon=${longitude}`,
      );

      location = data.display_name || `${latitude}, ${longitude}`;
    } catch (err) {
      console.error("Geolocation error:", err);

      if (err.code === 1) {
        error = "Location access denied. Please enable location permissions.";
      } else {
        error = "Unable to get your location. Please try again.";
      }
    }
  };

  let handleSubmit = async () => {
    error = "";
    success = false;
    submitting = true;

    const data = {
      title,
      location,
      longitude: longitude || -81.3792,
      latitude: latitude || 28.5383,
      tags: [
        { name: zoningTag, classification: "Zoning" },
        { name: progressTag, classification: "Progress" },
      ],
      description,
    };

    try {
      const response = await api.post(`/entries/create-entry`, data);
      success = true;
    } catch (e) {
      console.error("Submit error:", e);
      if (!error) error = e.message || "An unknown error occurred";
    } finally {
      submitting = false;
    }
  };
</script>

<div
  class="flex min-h-screen items-start justify-center bg-gradient-to-b from-slate-50 to-white px-4 py-12"
>
  <div class="w-full max-w-2xl">
    <div class="mb-6 flex items-center gap-3">
      <Retvrn />
      <h1 class="text-2xl font-semibold text-slate-800">
        Add an entry — help your neighbor know what's up
      </h1>
    </div>

    <form
      onsubmit={handleSubmit}
      class="rounded-2xl border border-slate-100 bg-white p-6 shadow-lg sm:p-8"
    >
      <p class="mb-4 text-sm text-slate-500">
        Share a short title, location and details. Fields marked with <span
          class="font-medium">*</span
        > are required.
      </p>

      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <Input
            bind:value={title}
            id="title"
            label="Title *"
            placeholder="e.g. Road closed near Elm St."
            class="w-full"
            required
            aria-required="true"
          />
        </div>

        <div class="sm:col-span-2">
          <label
            for="location"
            class="mb-1 block text-sm font-medium text-slate-700"
          >
            Location *
          </label>
          <div class="flex flex-row items-start gap-2">
            <div class="flex-1">
              <DropdownTextfield
                {suggestionsHidden}
                suggestions={suggestions.map(
                  (suggestion) => suggestion.address,
                )}
                handleInput={handleLocationAutosuggestion}
                selectSuggestion={handleSelectSuggestion}
                bind:searchValue={location}
              />
            </div>
            <Button
              onClick={() => handleGetCurrentLocation()}
              text="Get Current Location"
            />
          </div>
        </div>

        <div class="sm:col-span-2">
          <Input
            bind:value={description}
            id="description"
            label="Description"
            placeholder="Add context, ETA, or contact info..."
            rows={4}
            class="w-full"
            textarea={true}
          />
        </div>

        <div>
          <SelectInput
            id="progressTagsSelect"
            label="Progress Tag *"
            bind:value={progressTag}
            options={Tags.filter(
              (tag) => tag.classification === "Progress",
            ).map((t) => t.name)}
            required={true}
            class="w-full"
            aria-required="true"
          />
        </div>

        <div>
          <SelectInput
            id="zoningTagsSelect"
            label="Zoning Tag *"
            bind:value={zoningTag}
            options={Tags.filter((tag) => tag.classification === "Zoning").map(
              (t) => t.name,
            )}
            required={true}
            class="w-full"
            aria-required="true"
          />
        </div>
      </div>

      <div class="mt-5 flex items-center justify-between gap-4">
        <div>
          {#if error}
            <p class="text-sm text-red-600" role="alert">{error}</p>
          {/if}
          {#if success}
            <p class="text-sm text-green-700" role="status">Saved ✓</p>
          {/if}
        </div>

        <div class="ml-auto flex items-center gap-3">
          <button
            type="submit"
            class="inline-flex items-center gap-2 rounded-xl bg-sky-600 px-4 py-2 font-medium
                   text-white shadow-sm hover:cursor-pointer hover:bg-sky-700 focus:outline-none focus-visible:ring-4
                   focus-visible:ring-sky-200 active:scale-98"
            aria-disabled={submitting}
            disabled={submitting}
          >
            {#if submitting}
              <svg
                class="h-4 w-4 animate-spin"
                viewBox="0 0 24 24"
                fill="none"
                aria-hidden
              >
                <circle
                  cx="12"
                  cy="12"
                  r="10"
                  stroke="currentColor"
                  stroke-width="4"
                  class="opacity-25"
                ></circle>
                <path
                  d="M4 12a8 8 0 018-8"
                  stroke="currentColor"
                  stroke-width="4"
                  class="opacity-75"
                ></path>
              </svg>
              Saving...
            {:else}
              Submit
            {/if}
          </button>

          <button
            type="button"
            class="rounded-lg border border-slate-200 px-3 py-2 text-sm text-slate-700 hover:cursor-pointer hover:bg-slate-50"
            onclick={() => {
              title = "";
              location = "";
              description = "";
              progressTag = "";
              zoningTag = "";
              error = "";
              success = false;
            }}
          >
            Reset
          </button>
        </div>
      </div>
    </form>
  </div>
</div>

<style>
  /* tiny visual tweak: slightly reduce active button scale */
  .active\:scale-98:active {
    transform: scale(0.98);
  }
</style>
