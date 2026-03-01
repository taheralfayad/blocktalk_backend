<script>
  let { 
    onClick,
    text
  } = $props();
</script>

<button
  onclick={onClick}
  type="button"
  class="inline-flex items-center gap-2 px-4 py-2 rounded-xl font-medium shadow-sm
          bg-sky-600 text-white hover:bg-sky-700 active:scale-98 focus:outline-none focus-visible:ring-4
          focus-visible:ring-sky-200 hover:cursor-pointer"
>
  {text}
</button>

