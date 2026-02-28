<script lang="ts">
  import LeaseDetail from '$gen/components/entities/lease/LeaseDetail.svelte';

  export let id: string;
</script>

<div class="space-y-4">
  <div class="flex items-center gap-2">
    <a href="#/leases" class="btn btn-sm variant-soft">&larr; Back to Leases</a>
  </div>
  <LeaseDetail {id} />
</div>
