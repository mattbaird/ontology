<script lang="ts">
  import PersonDetail from '$gen/components/entities/person/PersonDetail.svelte';

  export let id: string;
</script>

<div class="space-y-4">
  <div class="flex items-center gap-2">
    <a href="#/persons" class="btn btn-sm variant-soft">&larr; Back to Persons</a>
  </div>
  <PersonDetail {id} />
</div>
