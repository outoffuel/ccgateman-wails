<script>
  import KioskSwipe from './lib/components/KioskSwipe.svelte';
  import AdminDashboard from './lib/components/AdminDashboard.svelte';
  import PINModal from './lib/components/PINModal.svelte';

  let currentView = 'kiosk'; // 'kiosk' | 'admin'
  let isPinModalOpen = false;

  function handleRequestOpenAdmin() {
    isPinModalOpen = true;
  }

  function handlePinSuccess() {
    isPinModalOpen = false;
    currentView = 'admin';
  }

  function handlePinCancel() {
    isPinModalOpen = false;
  }

  function handleBackToKiosk() {
    currentView = 'kiosk';
  }
</script>

<main class="w-full min-h-screen overflow-y-auto select-none bg-slate-950">
  {#if currentView === 'kiosk'}
    <KioskSwipe onOpenAdmin={handleRequestOpenAdmin} />
  {:else if currentView === 'admin'}
    <AdminDashboard onBackToKiosk={handleBackToKiosk} />
  {/if}

  {#if isPinModalOpen}
    <PINModal 
      onSuccess={handlePinSuccess}
      onCancel={handlePinCancel}
    />
  {/if}
</main>

<style>
  :global(html, body) {
    margin: 0;
    padding: 0;
    width: 100%;
    height: 100%;
    overflow-y: auto;
    background-color: #0f172a;
  }
</style>
