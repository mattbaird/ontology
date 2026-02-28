<script lang="ts">
  import BuildingList from '$gen/components/entities/building/BuildingList.svelte';
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between">
    <h1 class="h2">Buildings</h1>
    <a href="#/buildings/new" class="btn variant-filled-primary">New Building</a>
  </div>
  <BuildingList />
</div>
