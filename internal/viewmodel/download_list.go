package viewmodel

type DownloadList struct {
	root    *ViewModel
	service DownloadService
	Files   []*FileItem
}
