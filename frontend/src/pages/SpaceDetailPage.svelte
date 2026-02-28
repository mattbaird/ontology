<script lang="ts">
  import SpaceDetail from '$gen/components/entities/space/SpaceDetail.svelte';

  export let id: string;
</script>

<div class="space-y-4">
  <div class="flex items-center gap-2">
    <a href="#/spaces" class="btn btn-sm variant-soft">&larr; Back to Spaces</a>
  </div>
  <SpaceDetail {id} />
</div>
