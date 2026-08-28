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

<main class="w-full h-screen overflow-hidden select-none">
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
  :global(body) {
    margin: 0;
    padding: 0;
    overflow: hidden;
    background-color: #0f172a;
  }
</style>
