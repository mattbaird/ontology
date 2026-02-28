<script lang="ts">
  import PortfolioDetail from '$gen/components/entities/portfolio/PortfolioDetail.svelte';
  export let id: string;
</script>

<div class="space-y-4">
  <a href="#/portfolios" class="btn btn-sm variant-soft">&larr; Back to Portfolios</a>
  <PortfolioDetail {id} />
</div>
