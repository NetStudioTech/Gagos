// GAGOS - Lightweight DevOps Platform
// Main Application Entry Point

// API Base URL
export const API_BASE = '/api/v1';

// Import all modules
import { loadState, saveState, restoreState, setStateHelpers, setDesktopIconsGetter } from './state.js';
import {
    openWindow, closeWindow, minimizeWindow, maximizeWindow, bringToFront,
    startDrag, startResize, toggleStartMenu, updateTaskbar, setWindowOpenCallback, getActiveWindow
} from './windows.js';
import {
    DESKTOP_ICONS, loadDesktopPreferences, saveDesktopPreferences, resetDesktopPreferences,
    renderDesktopIcons, showDesktopContextMenu, hideDesktopContextMenu, toggleIconEditMode,
    hideIcon, showIcon
} from './desktop.js';
import { updateClock, checkHealth, logout } from './taskbar.js';
import {
    showNetTab, runPing, runDNS, runPortCheck, runTraceroute, runTelnet,
    runWhois, runSSLCheck, runCurl, loadInterfaces
} from './network.js';
import {
    showK8sTab, loadK8sData, loadNamespaces, loadPods, loadNodes, loadServices,
    loadDeployments, loadDaemonSets, loadStatefulSets, loadJobs, loadCronJobs,
    loadConfigMaps, loadSecrets, loadIngresses, loadPVCs, loadEvents,
    loadPodsForNamespace, loadServicesForNamespace, loadDeploymentsForNamespace,
    loadDaemonSetsForNamespace, loadStatefulSetsForNamespace, loadJobsForNamespace,
    loadCronJobsForNamespace, loadConfigMapsForNamespace, loadSecretsForNamespace,
    loadIngressesForNamespace, loadPVCsForNamespace, loadEventsForNamespace,
    toggleEditMode, enableEditMode, disableEditMode, showModal, closeModal,
    describeResource, decodeDescribedSecret, editResource, saveResourceEdit, showDeleteModal, confirmDelete,
    viewPodLogs, refreshLogs, showScaleModal, confirmScale, showRestartModal, confirmRestart,
    openCreateModal, loadResourceTemplate, createResource,
    toggleAutoRefresh, updateRefreshInterval
} from './kubernetes.js';
import { initTerminal, reconnectTerminal } from './terminal.js';
import {
    showCicdTab, loadCicdData, loadCicdStats, loadCicdPipelines, loadCicdRuns, loadCicdArtifacts,
    loadSamplePipeline, validatePipeline, createPipeline, triggerPipeline, viewPipeline,
    deletePipeline, clonePipeline, cancelRun, retryPipelineRun, viewRunJobs, closeCicdLogModal, deleteArtifact,
    copyPipelineBadge, copyFreestyleJobBadge,
    showAddSSHHostModal, closeSSHHostModal, updateAuthFields, saveSSHHost,
    showCreateFreestyleJobModal, closeFreestyleJobModal, saveFreestyleJob, addBuildStep,
    closeBuildConsole,
    showAddGitCredentialModal, closeGitCredentialModal, updateGitAuthFields, saveGitCredential
} from './cicd.js';
import { initNotepad, addNotepadTab, closeNotepadTab, saveNotepadContent, switchNotepadTab, renameNotepadTab } from './notepad.js';
import {
    showMonitoringTab, loadMonitoringData, loadMonitoringSummary, loadMonitoringNodes,
    loadMonitoringPods, loadMonitoringQuotas, loadMonitoringHPA
} from './monitoring.js';
import {
    showDevToolsTab, doBase64Encode, doBase64Decode, decodeK8sSecret, generateHashes,
    copyHashValue, compareHashes, checkCertificate, parseCertificate, generateSSHKey,
    validateSSHKey, convertFormat, formatJSON, minifyJSON, computeDiff, copyToClipboard
} from './devtools.js';
import {
    showPgTab, pgConnect, pgLoadInfo, pgExecuteQuery, pgDump, pgCopyDump, pgDownloadDump,
    showRedisTab, redisConnect, redisLoadInfo, redisScanKeys, redisGetKey, redisExecCommand, redisLoadCluster,
    showMysqlTab, mysqlConnect, mysqlLoadInfo, mysqlExecuteQuery, mysqlDump, mysqlCopyDump, mysqlDownloadDump
} from './database.js';
import {
    showS3Tab, s3Connect, s3LoadBuckets, s3CreateBucket, s3DeleteBucket, s3SelectBucket,
    s3LoadObjects, s3NavigateFolder, s3GoUp, s3UploadFiles, s3HandleFileSelect,
    s3DownloadFile, s3DeleteFile, s3GetInfo, s3GetPresignedURL, s3LoadPreset,
    s3CloseModal, s3CopyPresignedURL
} from './s3.js';
import {
    showEsTab, esConnect, esLoadPreset, esLoadIndices, esCreateIndex, esDeleteIndex,
    esRefreshIndex, esViewMapping, esViewSettings, esSelectIndex, esSearchDocuments,
    esViewDocument, esDeleteDocument, esExecuteQuery, esCloseModal, esCopyModalContent
} from './elasticsearch.js';
import { escapeHtml, formatDuration, formatSize, formatTime } from './utils.js';

// Set up circular dependency helpers
setDesktopIconsGetter(() => DESKTOP_ICONS);
setStateHelpers({
    getActiveWindow,
    showNetTab,
    showK8sTab
});

// Set up window open callbacks for initializing data
setWindowOpenCallback('kubernetes', loadK8sData);
setWindowOpenCallback('terminal', initTerminal);
setWindowOpenCallback('cicd', loadCicdData);
setWindowOpenCallback('monitoring', loadMonitoringData);
setWindowOpenCallback('network', loadInterfaces);
setWindowOpenCallback('notepad', initNotepad);

// Attach functions to window object for onclick handlers in HTML
// Window management
window.openWindow = openWindow;
window.closeWindow = closeWindow;
window.minimizeWindow = minimizeWindow;
window.maximizeWindow = maximizeWindow;
window.bringToFront = bringToFront;
window.startDrag = startDrag;
window.startResize = startResize;
window.toggleStartMenu = toggleStartMenu;

// Desktop icons
window.hideIcon = hideIcon;
window.showIcon = showIcon;
window.showDesktopContextMenu = showDesktopContextMenu;
window.hideDesktopContextMenu = hideDesktopContextMenu;
window.toggleIconEditMode = toggleIconEditMode;
window.saveDesktopPreferences = saveDesktopPreferences;
window.resetDesktopPreferences = resetDesktopPreferences;

// Taskbar
window.logout = logout;

// Network tools
window.showNetTab = showNetTab;
window.runPing = runPing;
window.runDNS = runDNS;
window.runPortCheck = runPortCheck;
window.runTraceroute = runTraceroute;
window.runTelnet = runTelnet;
window.runWhois = runWhois;
window.runSSLCheck = runSSLCheck;
window.runCurl = runCurl;
window.loadInterfaces = loadInterfaces;

// Kubernetes
window.showK8sTab = showK8sTab;
window.loadK8sData = loadK8sData;
window.loadPodsForNamespace = loadPodsForNamespace;
window.loadServicesForNamespace = loadServicesForNamespace;
window.loadDeploymentsForNamespace = loadDeploymentsForNamespace;
window.loadDaemonSetsForNamespace = loadDaemonSetsForNamespace;
window.loadStatefulSetsForNamespace = loadStatefulSetsForNamespace;
window.loadJobsForNamespace = loadJobsForNamespace;
window.loadCronJobsForNamespace = loadCronJobsForNamespace;
window.loadConfigMapsForNamespace = loadConfigMapsForNamespace;
window.loadSecretsForNamespace = loadSecretsForNamespace;
window.loadIngressesForNamespace = loadIngressesForNamespace;
window.loadPVCsForNamespace = loadPVCsForNamespace;
window.loadEventsForNamespace = loadEventsForNamespace;
window.toggleEditMode = toggleEditMode;
window.enableEditMode = enableEditMode;
window.showModal = showModal;
window.closeModal = closeModal;
window.describeResource = describeResource;
window.decodeDescribedSecret = decodeDescribedSecret;
window.editResource = editResource;
window.saveResourceEdit = saveResourceEdit;
window.showDeleteModal = showDeleteModal;
window.confirmDelete = confirmDelete;
window.viewPodLogs = viewPodLogs;
window.refreshLogs = refreshLogs;
window.showScaleModal = showScaleModal;
window.confirmScale = confirmScale;
window.showRestartModal = showRestartModal;
window.confirmRestart = confirmRestart;
window.openCreateModal = openCreateModal;
window.loadResourceTemplate = loadResourceTemplate;
window.createResource = createResource;
window.toggleAutoRefresh = toggleAutoRefresh;
window.updateRefreshInterval = updateRefreshInterval;

// Terminal
window.reconnectTerminal = reconnectTerminal;

// CI/CD
window.showCicdTab = showCicdTab;
window.loadSamplePipeline = loadSamplePipeline;
window.validatePipeline = validatePipeline;
window.createPipeline = createPipeline;
window.triggerPipeline = triggerPipeline;
window.viewPipeline = viewPipeline;
window.deletePipeline = deletePipeline;
window.clonePipeline = clonePipeline;
window.cancelRun = cancelRun;
window.retryPipelineRun = retryPipelineRun;
window.viewRunJobs = viewRunJobs;
window.closeCicdLogModal = closeCicdLogModal;
window.deleteArtifact = deleteArtifact;
window.copyPipelineBadge = copyPipelineBadge;
window.copyFreestyleJobBadge = copyFreestyleJobBadge;

// Freestyle Jobs & SSH Hosts
window.showAddSSHHostModal = showAddSSHHostModal;
window.closeSSHHostModal = closeSSHHostModal;
window.updateAuthFields = updateAuthFields;
window.saveSSHHost = saveSSHHost;
window.showCreateFreestyleJobModal = showCreateFreestyleJobModal;
window.closeFreestyleJobModal = closeFreestyleJobModal;
window.saveFreestyleJob = saveFreestyleJob;
window.addBuildStep = addBuildStep;
window.closeBuildConsole = closeBuildConsole;

// Git Credentials
window.showAddGitCredentialModal = showAddGitCredentialModal;
window.closeGitCredentialModal = closeGitCredentialModal;
window.updateGitAuthFields = updateGitAuthFields;
window.saveGitCredential = saveGitCredential;

// Notepad
window.addNotepadTab = addNotepadTab;
window.closeNotepadTab = closeNotepadTab;
window.saveNotepadContent = saveNotepadContent;
window.switchNotepadTab = switchNotepadTab;
window.renameNotepadTab = renameNotepadTab;

// Monitoring
window.showMonitoringTab = showMonitoringTab;
window.loadMonitoringPods = loadMonitoringPods;
window.loadMonitoringQuotas = loadMonitoringQuotas;
window.loadMonitoringHPA = loadMonitoringHPA;

// Dev Tools
window.showDevToolsTab = showDevToolsTab;
window.doBase64Encode = doBase64Encode;
window.doBase64Decode = doBase64Decode;
window.decodeK8sSecret = decodeK8sSecret;
window.generateHashes = generateHashes;
window.copyHashValue = copyHashValue;
window.compareHashes = compareHashes;
window.checkCertificate = checkCertificate;
window.parseCertificate = parseCertificate;
window.generateSSHKey = generateSSHKey;
window.validateSSHKey = validateSSHKey;
window.convertFormat = convertFormat;
window.formatJSON = formatJSON;
window.minifyJSON = minifyJSON;
window.computeDiff = computeDiff;
window.copyToClipboard = copyToClipboard;

// Database - PostgreSQL
window.showPgTab = showPgTab;
window.pgConnect = pgConnect;
window.pgExecuteQuery = pgExecuteQuery;
window.pgDump = pgDump;
window.pgCopyDump = pgCopyDump;
window.pgDownloadDump = pgDownloadDump;

// Database - Redis
window.showRedisTab = showRedisTab;
window.redisConnect = redisConnect;
window.redisScanKeys = redisScanKeys;
window.redisGetKey = redisGetKey;
window.redisExecCommand = redisExecCommand;
window.redisLoadCluster = redisLoadCluster;

// Database - MySQL
window.showMysqlTab = showMysqlTab;
window.mysqlConnect = mysqlConnect;
window.mysqlExecuteQuery = mysqlExecuteQuery;
window.mysqlDump = mysqlDump;
window.mysqlCopyDump = mysqlCopyDump;
window.mysqlDownloadDump = mysqlDownloadDump;

// S3 Storage
window.showS3Tab = showS3Tab;
window.s3Connect = s3Connect;
window.s3LoadBuckets = s3LoadBuckets;
window.s3CreateBucket = s3CreateBucket;
window.s3DeleteBucket = s3DeleteBucket;
window.s3SelectBucket = s3SelectBucket;
window.s3LoadObjects = s3LoadObjects;
window.s3NavigateFolder = s3NavigateFolder;
window.s3GoUp = s3GoUp;
window.s3UploadFiles = s3UploadFiles;
window.s3HandleFileSelect = s3HandleFileSelect;
window.s3DownloadFile = s3DownloadFile;
window.s3DeleteFile = s3DeleteFile;
window.s3GetInfo = s3GetInfo;
window.s3GetPresignedURL = s3GetPresignedURL;
window.s3LoadPreset = s3LoadPreset;
window.s3CloseModal = s3CloseModal;
window.s3CopyPresignedURL = s3CopyPresignedURL;

// Elasticsearch
window.showEsTab = showEsTab;
window.esConnect = esConnect;
window.esLoadPreset = esLoadPreset;
window.esLoadIndices = esLoadIndices;
window.esCreateIndex = esCreateIndex;
window.esDeleteIndex = esDeleteIndex;
window.esRefreshIndex = esRefreshIndex;
window.esViewMapping = esViewMapping;
window.esViewSettings = esViewSettings;
window.esSelectIndex = esSelectIndex;
window.esSearchDocuments = esSearchDocuments;
window.esViewDocument = esViewDocument;
window.esDeleteDocument = esDeleteDocument;
window.esExecuteQuery = esExecuteQuery;
window.esCloseModal = esCloseModal;
window.esCopyModalContent = esCopyModalContent;

// Utility
window.escapeHtml = escapeHtml;

// Initialize application when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    // Restore state (window positions, etc.)
    restoreState();

    // Load desktop icons
    loadDesktopPreferences();

    // Start clock and health check
    updateClock();
    setInterval(updateClock, 1000);
    checkHealth();
    setInterval(checkHealth, 30000);

    // Set up context menu for desktop
    document.getElementById('desktop').addEventListener('contextmenu', showDesktopContextMenu);
    document.addEventListener('click', (e) => {
        if (!e.target.closest('#desktop-context-menu') && !e.target.closest('.context-menu')) {
            hideDesktopContextMenu();
        }
    });

    // Global key handlers
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            hideDesktopContextMenu();
        }
    });

    console.log('GAGOS initialized successfully');
});
