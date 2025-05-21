package myproc

import (
	"golang.org/x/sys/windows"
)

const (
	IMAGE_DIRECTORY_ENTRY_EXPORT       = 0
	IMAGE_DIRECTORY_ENTRY_IMPORT       = 1
	IMAGE_DIRECTORY_ENTRY_EXCEPTION    = 3
	IMAGE_DIRECTORY_ENTRY_BASERELOC    = 5
	IMAGE_DIRECTORY_ENTRY_TLS          = 9
	IMAGE_DIRECTORY_ENTRY_DELAY_IMPORT = 13
)

type IMAGE_DOS_HEADER struct {
	E_magic    uint16
	E_cblp     uint16
	E_cp       uint16
	E_crlc     uint16
	E_cparhdr  uint16
	E_minalloc uint16
	E_maxalloc uint16
	E_ss       uint16
	E_sp       uint16
	E_csum     uint16
	E_ip       uint16
	E_cs       uint16
	E_lfarlc   uint16
	E_ovno     uint16
	E_res      [4]uint16
	E_oemid    uint16
	E_oeminfo  uint16
	E_res2     [10]uint16
	E_lfanew   uint32
}

type IMAGE_NT_HEADERS struct {
	Signature      uint32
	FileHeader     IMAGE_FILE_HEADER
	OptionalHeader IMAGE_OPTIONAL_HEADER
}

type IMAGE_FILE_HEADER struct {
	Machine              uint16
	NumberOfSections     uint16
	TimeDateStamp        uint32
	PointerToSymbolTable uint32
	NumberOfSymbols      uint32
	SizeOfOptionalHeader uint16
	Characteristics      uint16
}

type IMAGE_DATA_DIRECTORY struct {
	VirtualAddress uint32
	Size           uint32
}

type IMAGE_OPTIONAL_HEADER struct {
	Magic                       uint16
	MajorLinkerVersion          uint8
	MinorLinkerVersion          uint8
	SizeOfCode                  uint32
	SizeOfInitializedData       uint32
	SizeOfUninitializedData     uint32
	AddressOfEntryPoint         uint32
	BaseOfCode                  uint32
	ImageBase                   uintptr
	SectionAlignment            uint32
	FileAlignment               uint32
	MajorOperatingSystemVersion uint16
	MinorOperatingSystemVersion uint16
	MajorImageVersion           uint16
	MinorImageVersion           uint16
	MajorSubsystemVersion       uint16
	MinorSubsystemVersion       uint16
	Win32VersionValue           uint32
	SizeOfImage                 uint32
	SizeOfHeaders               uint32
	CheckSum                    uint32
	Subsystem                   uint16
	DllCharacteristics          uint16
	SizeOfStackReserve          uint64
	SizeOfStackCommit           uint64
	SizeOfHeapReserve           uint64
	SizeOfHeapCommit            uint64
	LoaderFlags                 uint32
	NumberOfRvaAndSizes         uint32
	DataDirectory               [16]IMAGE_DATA_DIRECTORY
}

type IMAGE_SECTION_HEADER struct {
	Name                 [8]byte
	VirtualSize          uint32
	VirtualAddress       uint32
	SizeOfRawData        uint32
	PointerToRawData     uint32
	PointerToRelocations uint32
	PointerToLinenumbers uint32
	NumberOfRelocations  uint16
	NumberOfLinenumbers  uint16
	Characteristics      uint32
}

type IMAGE_EXPORT_DIRECTORY struct {
	Characteristics       uint32
	TimeDateStamp         uint32
	MajorVersion          uint16
	MinorVersion          uint16
	Name                  uint32
	Base                  uint32
	NumberOfFunctions     uint32
	NumberOfNames         uint32
	AddressOfFunctions    uint32
	AddressOfNames        uint32
	AddressOfNameOrdinals uint32
}

type PEB struct {
	InheritedAddressSpace    byte
	ReadImageFileExecOptions byte
	BeingDebugged            byte
	BitField                 byte
	/* BitField:
		union
	    {
	        BOOLEAN BitField;
	        struct
	        {
	            BOOLEAN ImageUsesLargePages : 1;
	            BOOLEAN IsProtectedProcess : 1;
	            BOOLEAN IsImageDynamicallyRelocated : 1;
	            BOOLEAN SkipPatchingUser32Forwarders : 1;
	            BOOLEAN IsPackagedProcess : 1;
	            BOOLEAN IsAppContainer : 1;
	            BOOLEAN IsProtectedProcessLight : 1;
	            BOOLEAN IsLongPathAwareProcess : 1;
	        };
	    };
	*/
	Mutant            uintptr
	ImageBaseAddress  uintptr
	Ldr               *PEB_LDR_DATA
	ProcessParameters *RTL_USER_PROCESS_PARAMETERS
	SubSystemData     uintptr
	ProcessHeap       uintptr
	FastPebLock       uintptr // RTL_CRITICAL_SECTION*
	AtlThunkSListPtr  uintptr // SLIST_HEADER*
	IFEOKey           uintptr
	CrossProcessFlags uint32
	/* CrossProcessFlags:
		union
	    {
	        ULONG CrossProcessFlags;
	        struct
	        {
	            ULONG ProcessInJob : 1;
	            ULONG ProcessInitializing : 1;
	            ULONG ProcessUsingVEH : 1;
	            ULONG ProcessUsingVCH : 1;
	            ULONG ProcessUsingFTH : 1;
	            ULONG ProcessPreviouslyThrottled : 1;
	            ULONG ProcessCurrentlyThrottled : 1;
	            ULONG ProcessImagesHotPatched : 1; // REDSTONE5
	            ULONG ReservedBits0 : 24;
	        };
	    };
	*/
	Data uintptr
	/* Data:
		union
	    {
	        PVOID KernelCallbackTable;
	        PVOID UserSharedInfoPtr;
	    };
	*/
	SystemReserved                 uint32
	AtlThunkSListPtr32             uint32
	ApiSetMap                      *API_SET_NAMESPACE // API_SET_NAMESPACE*
	TlsExpansionCounter            uint32
	TlsBitmap                      uintptr // PRTL_BITMAP
	TlsBitmapBits                  [2]uint32
	ReadOnlySharedMemoryBase       uintptr
	SharedData                     uintptr // HotpatchInformation
	ReadOnlyStaticServerData       uintptr
	AnsiCodePageData               uintptr // CPTABLEINFO*
	OemCodePageData                uintptr // CPTABLEINFO*
	UnicodeCaseTable               uintptr // NLSTABLEINFO*
	NumberOfProcessors             uint32
	NtGlobalFlag                   uint32
	CriticalSectionTimeout         uint64
	HeapSegmentReserve             uint64
	HeapSegmentCommit              uint64
	HeapDeCommitTotalFreeThreshold uint64
	HeapDeCommitFreeBlockThreshold uint64
	NumberOfHeaps                  uint32
	MaximumNumberOfHeaps           uint32
	ProcessHeaps                   uintptr // HEAP**
	GdiSharedHandleTable           uintptr
	ProcessStarterHelper           uintptr
	GdiDCAttributeList             uint32
	LoaderLock                     uintptr // RTL_CRITICAL_SECTION*
	OSMajorVersion                 uint32
	OSMinorVersion                 uint32
	OSBuildNumber                  uint16
	OSCSDVersion                   uint16
	OSPlatformId                   uint32
	ImageSubsystem                 uint32
	ImageSubsystemMajorVersion     uint32
	ImageSubsystemMinorVersion     uint32
	ActiveProcessAffinityMask      uint64
	GdiHandleBuffer                [60]uint32
	PostProcessInitRoutine         uintptr
	TlsExpansionBitmap             uintptr
	TlsExpansionBitmapBits         [32]uint32
	SessionId                      uint32
	AppCompatFlags                 uint64
	AppCompatFlagsUser             uint64
	ShimData                       uintptr
	AppCompatInfo                  uintptr // APPCOMPAT_EXE_DATA
	CSDVersion                     windows.NTUnicodeString
	ActivationContextData          uintptr // ACTIVATION_CONTEXT_DATA
	ProcessAssemblyStorageMap      uintptr // ASSEMBLY_STORAGE_MAP
	SystemDefaultActivationContext uintptr // ACTIVATION_CONTEXT_DATA
	SystemAssemblyStorageMap       uintptr // ASSEMBLY_STORAGE_MAP
	MinimumStackCommit             uint64
	SparePointers                  [4]uintptr // 19H1 (previously FlsCallback to FlsHighIndex)
	SpareUlongs                    [5]uint32  // 19H1
	/*
			PVOID* FlsCallback;
		    LIST_ENTRY FlsListHead;
		    PVOID FlsBitmap;
		    ULONG FlsBitmapBits[FLS_MAXIMUM_AVAILABLE / (sizeof(ULONG) * 8)];
		    ULONG FlsHighIndex;
	*/
	WerRegistrationData uintptr
	WerShipAssertPtr    uintptr
	Unused              uintptr
	ImageHeaderHash     uintptr
	TracingFlags        uint32
	/* TracingFlags:
		union
	    {
	        ULONG TracingFlags;
	        struct
	        {
	            ULONG HeapTracingEnabled : 1;
	            ULONG CritSecTracingEnabled : 1;
	            ULONG LibLoaderTracingEnabled : 1;
	            ULONG SpareTracingBits : 29;
	        };
	    };
	*/
	CsrServerReadOnlySharedMemoryBase    uintptr
	TppWorkerpListLock                   uintptr // RTL_CRITICAL_SECTION*
	TppWorkerpList                       *LIST_ENTRY
	WaitOnAddressHashTable               [128]uintptr
	TelemetryCoverageHeader              uintptr // REDSTONE3
	CloudFileFlags                       uint32
	CloudFileDiagInfo                    uint32 // REDSTONE4
	PlaceholderCompatibilityMode         uint8
	PlaceholderCompatibilityModeReserved [7]uint8
	LeapSecondData                       uintptr // REDSTONE5
	LeapSecondFlags                      uint32
	/* LeapSecondFlags:
		union
	    {
	        ULONG LeapSecondFlags;
	        struct
	        {
	            ULONG SixtySecondEnabled : 1;
	            ULONG Reserved : 31;
	        };
	    };
	*/
	NtGlobalFlag2 uint32
}

type PEB_LDR_DATA struct {
	Length                          uint32
	Initialized                     uint8
	SsHandle                        uintptr
	InLoadOrderModuleList           LIST_ENTRY
	InMemoryOrderModuleList         LIST_ENTRY
	InInitializationOrderModuleList LIST_ENTRY
	EntryInProgress                 uintptr
	ShutdownInProgress              uint8
	ShutdownThreadId                uintptr
}

type LIST_ENTRY struct {
	Flink *LIST_ENTRY
	Blink *LIST_ENTRY
}

type RTL_USER_PROCESS_PARAMETERS struct {
	MaximumLength                    uint32
	Length                           uint32
	Flags                            uint32
	DebugFlags                       uint32
	ConsoleHandle                    windows.Handle
	ConsoleFlags                     uint32
	StandardInput                    windows.Handle
	StandardOutput                   windows.Handle
	StandardError                    windows.Handle
	CurrentDirectory                 CURDIR
	DllPath                          windows.NTUnicodeString
	ImagePathName                    windows.NTUnicodeString
	CommandLine                      windows.NTUnicodeString
	Environment                      uintptr
	StartingX                        uint32
	StartingY                        uint32
	CountX                           uint32
	CountY                           uint32
	CountCharsX                      uint32
	CountCharsY                      uint32
	FillAttribute                    uint32
	WindowFlags                      uint32
	ShowWindowFlags                  uint32
	WindowTitle                      windows.NTUnicodeString
	DesktopInfo                      windows.NTUnicodeString
	ShellInfo                        windows.NTUnicodeString
	RuntimeData                      windows.NTUnicodeString
	CurrentDirectories               [32]RTL_DRIVER_LETTER_CURDIR
	EnvironmentSize                  uintptr
	EnvironmentVersion               uintptr
	PackageDependencyData            uintptr
	ProcessGroUpId                   uint32
	LoaderThreads                    uint32
	RedirectionDllName               windows.NTUnicodeString
	HeapPartitionName                windows.NTUnicodeString
	DefaultThreadpoolCpuSetMasks     uintptr
	DefaultThreadpoolCpuSetMaskCount uint32
}

type CURDIR struct {
	DosPath windows.NTUnicodeString
	Handle  windows.Handle
}

type RTL_DRIVER_LETTER_CURDIR struct {
	Flags     uint16
	Length    uint16
	TimeStamp uint64
}

type API_SET_NAMESPACE struct {
	Version     uint32
	Size        uint32
	Flags       uint32
	Count       uint32
	EntryOffset uint32
	HashOffset  uint32
	HashFactor  uint32
}

type API_SET_HASH_ENTRY struct {
	Hash  uint32
	Index uint32
}

type API_SET_NAMESPACE_ENTRY struct {
	Flags        uint32
	NameOffset   uint32
	NameLength   uint32
	HashedLength uint32
	ValueOffset  uint32
	ValueCount   uint32
}

type API_SET_VALUE_ENTRY struct {
	Flags       uint32
	NameOffset  uint32
	NameLength  uint32
	ValueOffset uint32
	ValueLength uint32
}

type LDR_DATA_TABLE_ENTRY struct {
	InLoadOrderLinks           LIST_ENTRY
	InMemoryOrderLinks         LIST_ENTRY
	InInitializationOrderLinks LIST_ENTRY
	/* Union1:
		union
	    {
	        LIST_ENTRY InInitializationOrderLinks;
	        LIST_ENTRY InProgressLinks;
	    };
	*/
	DllBase     uintptr
	EntryPoint  uintptr
	SizeOfImage uint32
	FullDllName windows.NTUnicodeString
	BaseDllName windows.NTUnicodeString
	Flags       uint32
	/* Flags:
		union
	    {
	        UCHAR FlagGroup[4];
	        ULONG Flags;
	        struct
	        {
	            ULONG PackagedBinary : 1;
	            ULONG MarkedForRemoval : 1;
	            ULONG ImageDll : 1;
	            ULONG LoadNotificationsSent : 1;
	            ULONG TelemetryEntryProcessed : 1;
	            ULONG ProcessStaticImport : 1;
	            ULONG InLegacyLists : 1;
	            ULONG InIndexes : 1;
	            ULONG ShimDll : 1;
	            ULONG InExceptionTable : 1;
	            ULONG ReservedFlags1 : 2;
	            ULONG LoadInProgress : 1;
	            ULONG LoadConfigProcessed : 1;
	            ULONG EntryProcessed : 1;
	            ULONG ProtectDelayLoad : 1;
	            ULONG ReservedFlags3 : 2;
	            ULONG DontCallForThreads : 1;
	            ULONG ProcessAttachCalled : 1;
	            ULONG ProcessAttachFailed : 1;
	            ULONG CorDeferredValidate : 1;
	            ULONG CorImage : 1;
	            ULONG DontRelocate : 1;
	            ULONG CorILOnly : 1;
	            ULONG ChpeImage : 1;
	            ULONG ReservedFlags5 : 2;
	            ULONG Redirected : 1;
	            ULONG ReservedFlags6 : 2;
	            ULONG CompatDatabaseProcessed : 1;
	        };
	    };
	*/
	ObsoleteLoadCount           uint16
	TlsIndex                    uint16
	HashLinks                   LIST_ENTRY
	TimeDateStamp               uint32
	EntryPointActivationContext uintptr // *ACTIVATION_CONTEXT
	Lock                        uintptr
	DdagNode                    *LDR_DDAG_NODE
	NodeModuleLink              LIST_ENTRY
	LoadContext                 uintptr // _LDRP_LOAD_CONTEXT *
	ParentDllBase               uintptr
	SwitchBackContext           uintptr
	BaseAddressIndexNode        RTL_BALANCED_NODE
	MappingInfoIndexNode        RTL_BALANCED_NODE
	OriginalBase                uintptr
	LoadTime                    int64
	BaseNameHashValue           uint32
	LoadReason                  uint32
	ImplicitPathOptions         uint32
	ReferenceCount              uint32
	DependentLoadFlags          uint32
	SigningLevel                byte
}

type LDR_DDAG_NODE struct {
	Modules                 LIST_ENTRY
	ServiceTagList          *LDR_SERVICE_TAG_RECORD
	LoadCount               uint32
	LoadWhileUnloadingCount uint32
	LowestLink              uint32
	RemovalLink             SINGLE_LIST_ENTRY
	/* Union1:
		union {
	            LDRP_CSLIST Dependencies;
	            SINGLE_LIST_ENTRY RemovalLink;
	   };
	*/
	IncomingDependencies LDRP_CSLIST
	State                int
	CondenseLink         LIST_ENTRY
	PreorderNumber       uint32
}

type LDR_SERVICE_TAG_RECORD struct {
	Next       *LDR_SERVICE_TAG_RECORD
	ServiceTag uint32
}

type LDRP_CSLIST struct {
	Tail *SINGLE_LIST_ENTRY
}

type SINGLE_LIST_ENTRY struct {
	Next *SINGLE_LIST_ENTRY
}

type RTL_BALANCED_NODE struct {
	Left  *RTL_BALANCED_NODE
	Right *RTL_BALANCED_NODE
	/* Children:
		union
	    {
	        struct _RTL_BALANCED_NODE* Children[2];      //0x0
	        struct
	        {
	            struct _RTL_BALANCED_NODE* Left;         //0x0
	            struct _RTL_BALANCED_NODE* Right;        //0x8
	        };
	    }
	*/
	Data uintptr
	/* Data:
		union
	    {
	        struct
	        {
	            UCHAR Red:1;                             //0x10
	            UCHAR Balance:2;                         //0x10
	        };
	        ULONGLONG ParentValue;                       //0x10
	    }
	*/
}

type TEB struct {
	NtTib                              NT_TIB
	EnvironmentPointer                 uintptr
	ClientId                           CLIENT_ID
	ActiveRpcHandle                    uintptr
	ThreadLocalStoragePointer          uintptr
	ProcessEnvironmentBlock            *PEB
	LastErrorValue                     uint32
	CountOfOwnedCriticalSections       uint32
	CsrClientThread                    uintptr
	Win32ThreadInfo                    uintptr
	User32Reserved                     [26]uint32
	UserReserved                       [5]uint32
	WOW32Reserved                      uintptr
	CurrentLocale                      uint32
	FpSoftwareStatusRegister           uint32
	ReservedForDebuggerInstrumentation [16]uintptr
	SystemReserved1                    [30]uintptr
	PlaceholderCompatibilityMode       byte
	PlaceholderHydrationAlwaysExplicit uint8
	PlaceholderReserved                [10]byte
	ProxiedProcessId                   uint32
	ActivationStack                    ACTIVATION_CONTEXT_STACK
	WorkingOnBehalfTicket              [8]byte
	ExceptionCode                      uint32
	Padding0                           [4]byte
	ActivationContextStackPointer      *ACTIVATION_CONTEXT_STACK
	InstrumentationCallbackSp          uint64
	InstrumentationCallbackPreviousPc  uint64
	InstrumentationCallbackPreviousSp  uint64
	TxFsContext                        uint32
	InstrumentationCallbackDisabled    byte
	UnalignedLoadStoreExceptions       byte
	Padding1                           [2]byte
	GdiTebBatch                        GDI_TEB_BATCH
	RealClientId                       CLIENT_ID
	GdiCachedProcessHandle             uintptr
	GdiClientPID                       uint32
	GdiClientTID                       uint32
	GdiThreadLocalInfo                 uint32
	Win32ClientInfo                    [62]uint64
	glDispatchTable                    [233]uintptr
	glReserved1                        [29]uint64
	glReserved2                        uint32
	glSectionInfo                      uint32
	glSection                          uint32
	glTable                            uint32
	glCurrentRC                        uint32
	glContext                          uint32
	LastStatusValue                    uint32
	Padding2                           [4]byte
	StaticUnicodeString                windows.NTUnicodeString
	StaticUnicodeBuffer                [261]uint16
	Padding3                           [6]byte
	DeallocationStack                  uintptr
	TlsSlots                           [64]uintptr
	TlsLinks                           LIST_ENTRY
	Vdm                                uintptr
	ReservedForNtRpc                   uintptr
	DbgSsReserved                      [2]uintptr
	HardErrorMode                      uint32
	Padding4                           [4]byte
	Instrumentation                    [11]uintptr
	ActivityId                         windows.GUID
	SubProcessTag                      uintptr
	PerflibData                        uintptr
	EtwTraceData                       uintptr
	WinSockData                        uintptr
	GdiBatchCount                      uint32
	IdealProcessor                     uint32
	/*
			union
		    {
		        struct _PROCESSOR_NUMBER CurrentIdealProcessor;                     //0x1744
		        ULONG IdealProcessorValue;                                          //0x1744
		        struct
		        {
		            UCHAR ReservedPad0;                                             //0x1744
		            UCHAR ReservedPad1;                                             //0x1745
		            UCHAR ReservedPad2;                                             //0x1746
		            UCHAR IdealProcessor;                                           //0x1747
		        };
		    };
	*/
	GuaranteedStackBytes     uint32
	Padding5                 [4]byte
	ReservedForPerf          uintptr
	ReservedForOle           uintptr
	WaitingOnLoaderLock      uint32
	Padding6                 [4]byte
	SavedPriorityState       uintptr
	ReservedForCodeCoverage  uint64
	ThreadPoolData           uintptr
	TlsExpansionSlots        *uintptr
	ChpeV2CpuAreaInfo        uintptr // _CHPEV2_CPUAREA_INFO*
	Unused                   uintptr
	MuiGeneration            uint32
	IsImpersonating          uint32
	NlsCache                 uintptr
	ShimData                 uintptr
	HeapData                 uint32
	Padding7                 [4]byte
	CurrentTransactionHandle uintptr
	ActiveFrame              *TEB_ACTIVE_FRAME
	FlsData                  uintptr
	PreferredLanguages       uintptr
	UserPrefLanguages        uintptr
	MergedPrefLanguages      uintptr
	MuiImpersonation         uint32
	CrossTebFlag             uint16
	/*
			union
		    {
		        volatile USHORT CrossTebFlags;                                      //0x17ec
		        USHORT SpareCrossTebBits:16;                                        //0x17ec
		    };
	*/
	SameTebFlags uint16
	/*
			union
		    {
		        USHORT SameTebFlags;                                                //0x17ee
		        struct
		        {
		            USHORT SafeThunkCall:1;                                         //0x17ee
		            USHORT InDebugPrint:1;                                          //0x17ee
		            USHORT HasFiberData:1;                                          //0x17ee
		            USHORT SkipThreadAttach:1;                                      //0x17ee
		            USHORT WerInShipAssertCode:1;                                   //0x17ee
		            USHORT RanProcessInit:1;                                        //0x17ee
		            USHORT ClonedThread:1;                                          //0x17ee
		            USHORT SuppressDebugMsg:1;                                      //0x17ee
		            USHORT DisableUserStackWalk:1;                                  //0x17ee
		            USHORT RtlExceptionAttached:1;                                  //0x17ee
		            USHORT InitialThread:1;                                         //0x17ee
		            USHORT SessionAware:1;                                          //0x17ee
		            USHORT LoadOwner:1;                                             //0x17ee
		            USHORT LoaderWorker:1;                                          //0x17ee
		            USHORT SkipLoaderInit:1;                                        //0x17ee
		            USHORT SkipFileAPIBrokering:1;                                  //0x17ee
		        };
		    };
	*/
	TxnScopeEnterCallback      uintptr
	TxnScopeExitCallback       uintptr
	TxnScopeContext            uintptr
	LockCount                  uint32
	WowTebOffset               int32
	ResourceRetValue           uintptr
	ReservedForWdf             uintptr
	ReservedForCrt             uint64
	EffectiveContainerId       windows.GUID
	LastSleepCounter           uint64
	SpinCallCount              uint32
	Padding8                   [4]byte
	ExtendedFeatureDisableMask uint64
}

type NT_TIB struct {
	ExceptionList *EXCEPTION_REGISTRATION_RECORD
	StackBase     uintptr
	StackLimit    uintptr
	SubSystemTib  uintptr
	FiberData     uintptr
	/* Union :
		union
	    {
	        VOID* FiberData;                                                    //0x20
	        ULONG Version;                                                      //0x20
	    };
	*/
	ArbitraryUserPointer uintptr
	Self                 *NT_TIB
}

type EXCEPTION_REGISTRATION_RECORD struct {
	Next   *EXCEPTION_REGISTRATION_RECORD
	Handle uintptr
}

type CLIENT_ID struct {
	UniqueProcess uintptr
	UniqueThread  uintptr
}

type ACTIVATION_CONTEXT_STACK struct {
	ActiveFrame              *RTL_ACTIVATION_CONTEXT_STACK_FRAME
	FrameListCache           LIST_ENTRY
	Flags                    uint32
	NextCookieSequenceNumber uint32
	StackId                  uint32
}

type RTL_ACTIVATION_CONTEXT_STACK_FRAME struct {
	Previous          *RTL_ACTIVATION_CONTEXT_STACK_FRAME
	ActivationContext uintptr // _ACTIVATION_CONTEXT*
	Flags             uint32
}

type GDI_TEB_BATCH struct {
	OffsetAndHasRenderingCommand uint32
	HDC                          uint64
	Buffer                       [310]uint32
}

type TEB_ACTIVE_FRAME struct {
	Flags    uint32
	Previous *TEB_ACTIVE_FRAME
	Context  *TEB_ACTIVE_FRAME_CONTEXT
}

type TEB_ACTIVE_FRAME_CONTEXT struct {
	Flags     uint32
	FrameName *byte
}
