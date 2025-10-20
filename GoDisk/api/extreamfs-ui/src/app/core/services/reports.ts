import { HttpClient, HttpParams } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable, of, throwError, timer } from 'rxjs';
import { catchError, switchMap, map, tap, retryWhen, delay, take } from 'rxjs/operators';


export interface MBRPartReport {
  index: number;
  status: string;
  type: string;
  fit: string;
  rawStatus: number;
  rawType: number;
  rawFit: number;
  start: number;
  size: number;
  name: string;
  usable: boolean;
  id?: string;
  correlative?: number;
}

export interface MBRReport {
  kind: 'mbr';
  diskPath: string;
  created: string;
  sizeBytes: number;
  signature: number;
  fit: string;
  rawFit: number;
  partitions: MBRPartReport[];
}

export interface DiskSegment {
  kind: 'MBR'|'P'|'E'|'L'|'EBR'|'FREE';
  label: string;
  start: number;
  size: number;
  end: number;
  percent: number;   // 0..100
}
export interface ExtendedView {
  start: number;
  size: number;
  segments: DiskSegment[];
}
export interface DiskReport {
  kind: 'disk';
  diskPath: string;
  sizeBytes: number;
  mbrBytes: number;
  segments: DiskSegment[];
  extended?: ExtendedView;
}


export interface InodeReport {
  kind: 'inode';
  diskPath: string;
  id: string;
  index: number;
  type: string;
  rawType: number;
  size: number;
  uid: number;
  gid: number;
  perm: string;
  permRaw: number[];
  atime: string;
  mtime: string;
  ctime: string;
  blocksUsed: number;
  blocks: number[];
}

export interface InodeMini {
  index: number;
  type: string;
  rawType: number;
  size: number;
  blocksUsed: number;
}
export interface InodesReport {
  kind: 'inodes';
  diskPath: string;
  id: string;
  count: number;
  items: InodeMini[];
}

// ==== BLOCK REPORT ====
export interface DirEntry { name: string; inode: number; }
export interface DirBlockView { entries: DirEntry[]; }
export interface FileBlockView { size: number; preview: string; }
export interface PtrBlockView { pointers: number[]; }

export interface BlockItem {
  index: number;
  type: 'dir'|'file'|'ptr'|'unknown'|string;
  refCount: number;
  dir?: DirBlockView;
  file?: FileBlockView;
  ptr?: PtrBlockView;
}


export interface BlockReport {
  kind: 'block';
  diskPath: string;
  id: string;
  blockSize: number;
  count: number;
  used: number;
  blocks: BlockItem[];  
}

export interface PtrLevel { block: number; pointers: number[]; }
export interface PtrLevel2 { block: number; groups: PtrLevel[]; }
export interface BlocksView {
  direct: number[];
  indirect?: PtrLevel;
  doubleIndirect?: PtrLevel2;
}


export interface TreeBlockCard {
  index: number;
  type: 'dir' | 'file' | string;
  dir?: { entries: { name: string; inode: number }[] };
  file?: { size: number; preview?: string };
}

export interface TreeInode {
  index: number;
  type: string;
  rawType: number;
  size: number;
  uid: number;
  gid: number;
  perm: string;
  blocks: BlocksView;
  blocksFlat: number[];
  directCards?: TreeBlockCard[]; 
}
export interface TreeEdge {
  parent: number;
  name: string;
  child: number;
}
export interface TreeReport {
  kind: 'tree';
  diskPath: string;
  id: string;
  blockSize: number;
  inodes: number;
  blocks: number;
  root: number;
  nodes: TreeInode[];
  edges: TreeEdge[];
}

export interface SBReport {
  kind: 'sb';
  diskPath: string;
  id: string;

  blockSize: number;
  inodesCount: number;
  blocksCount: number;
  freeInodes: number;
  freeBlocks: number;
  inodeSize: number;
  bmInodeStart: number;
  bmBlockStart: number;
  inodeTableStart: number;
  blockStart: number;

  bitmapUsedInodes: number;
  bitmapFreeInodes: number;
  bitmapUsedBlocks: number;
  bitmapFreeBlocks: number;
}


export interface LSItem {
  name: string;
  type: string;     // "dir" | "file" | "other"
  rawType: number;
  inode: number;
  size: number;
  perm: string;
  uid: number;
  gid: number;
  owner?: string;
  group?: string;
  mtime: string;
  atime: string;
  ctime: string;
}

export interface LSReport {
  kind: 'ls';
  diskPath: string;
  id: string;
  dir: string;      
  items: LSItem[];
}

export interface JournalRow {
  count: number;
  operation: string;
  path: string;
  content: string;
  date: string; 
}


@Injectable({ providedIn: 'root' })
export class Reports {
  private http = inject(HttpClient);
  private base = '/api';

  getMBR(id: string) {
    return this.http.get<MBRReport>(`${this.base}/reports/mbr`, { params: { id, t: Date.now().toString() } });
  }


  getMBRWithRetry(id: string, tries = 2, delayMs = 250): Observable<MBRReport> {
    const attempt = (left: number): Observable<MBRReport> =>
      this.getMBR(id).pipe(
        switchMap(rep => {
          const ok = rep?.partitions?.length; 
          if (ok) return of(rep);
          if (left <= 0) return of(rep); // sin datos, sin reintentos
          return timer(delayMs).pipe(switchMap(() => attempt(left - 1)));
        }),
        catchError(err => {
          if (left <= 0) return throwError(() => err);
          return timer(delayMs).pipe(switchMap(() => attempt(left - 1)));
        })
      );
    return attempt(tries);
  }


  getDisk(id: string) {
    return this.http.get<DiskReport>(`${this.base}/reports/disk`, { params: { id, t: Date.now().toString() } });
  }


  getDiskWithRetry(id: string, tries = 2, delayMs = 250): Observable<DiskReport> {
    const attempt = (left: number): Observable<DiskReport> =>
      this.getDisk(id).pipe(
        switchMap(rep => {
          const ok = rep?.segments?.length;
          if (ok) return of(rep);
          if (left <= 0) return of(rep);
          return timer(delayMs).pipe(switchMap(() => attempt(left - 1)));
        }),
        catchError(err => {
          if (left <= 0) return throwError(() => err);
          return timer(delayMs).pipe(switchMap(() => attempt(left - 1)));
        })
      );
    return attempt(tries);
  }

  getInode(id: string, ruta: string) {
  return this.http.get<InodeReport>(`${this.base}/reports/inode`, {
    params: { id, ruta, t: Date.now().toString() },
  });
}

getInodeWithRetry(id: string, ruta: string, tries = 2, delayMs = 250): Observable<InodeReport> {
  const attempt = (left: number): Observable<InodeReport> =>
    this.getInode(id, ruta).pipe(
      switchMap(rep => {
        const ok = rep?.blocksUsed && rep.blocksUsed > 0;
        if (ok) return of(rep);
        if (left <= 0) return of(rep);
        return timer(delayMs).pipe(switchMap(() => attempt(left - 1)));
      }),
      catchError(err => {
        if (left <= 0) return throwError(() => err);
        return timer(delayMs).pipe(switchMap(() => attempt(left - 1)));
      })
    );
  return attempt(tries);
}

getInodes(id: string, max = 0) {
  const params: any = { id, t: Date.now().toString() };
  if (max > 0) params.max = String(max);
  return this.http.get<InodesReport>(`${this.base}/reports/inodes`, { params });
}

getBlocks(id: string) {
  return this.http.get<BlockReport>(`${this.base}/reports/block`, {
    params: { id, t: Date.now().toString() }
  });
}

getBlocksWithRetry(id: string, tries = 2, delayMs = 250): Observable<BlockReport> {
  const attempt = (left: number): Observable<BlockReport> =>
    this.getBlocks(id).pipe(
      switchMap((rep) => {
        const len =
          Array.isArray((rep as any).blocks) ? (rep as any).blocks.length :
          Array.isArray((rep as any).items)  ? (rep as any).items.length  : 0;

        if (len || left <= 0) return of(rep as BlockReport);
        return timer(delayMs).pipe(switchMap(() => attempt(left - 1)));
      }),
      catchError(err => {
        if (left <= 0) return throwError(() => err);
        return timer(delayMs).pipe(switchMap(() => attempt(left - 1)));
      })
    );

  return attempt(tries);
}

// --- BM_INODE ---
getBmInode(id: string) {
  return this.http.get(`/api/reports/bm_inode`, {
    params: { id, t: Date.now().toString() },
    responseType: 'text' as 'json'
  }) as unknown as Observable<string>;
}

getBmInodeWithRetry(id: string, tries = 2, delayMs = 250): Observable<string> {
  const attempt = (left: number): Observable<string> =>
    this.getBmInode(id).pipe(
      switchMap(txt => {
        const ok = !!(txt && String(txt).trim().length);
        if (ok || left <= 0) return of(String(txt));
        return timer(delayMs).pipe(switchMap(() => attempt(left - 1)));
      }),
      catchError(err => {
        if (left <= 0) return throwError(() => err);
        return timer(delayMs).pipe(switchMap(() => attempt(left - 1)));
      })
    );
  return attempt(tries);
}


getBmBlock(id: string) {
  return this.http.get(`/api/reports/bm_block`, {
    params: { id, t: Date.now().toString() },
    responseType: 'text' as 'json'
  }) as unknown as Observable<string>;
}

getBmBlockWithRetry(id: string, tries = 2, delayMs = 250): Observable<string> {
  const attempt = (left: number): Observable<string> =>
    this.getBmBlock(id).pipe(
      switchMap(txt => {
        const ok = !!(txt && String(txt).trim().length);
        if (ok || left <= 0) return of(String(txt));
        return timer(delayMs).pipe(switchMap(() => attempt(left - 1)));
      }),
      catchError(err => {
        if (left <= 0) return throwError(() => err);
        return timer(delayMs).pipe(switchMap(() => attempt(left - 1)));
      })
    );
  return attempt(tries);
}

// ===== API TREE =====
getTree(id: string) {
  return this.http.get<TreeReport>(`${this.base}/reports/tree`, {
    params: { id, t: Date.now().toString() }
  });
}
getTreeWithRetry(id: string, tries = 2, delayMs = 250): Observable<TreeReport> {
  const attempt = (left: number): Observable<TreeReport> =>
    this.getTree(id).pipe(
      switchMap(rep => {
        const ok = !!(rep?.nodes?.length);
        if (ok || left <= 0) return of(rep);
        return timer(delayMs).pipe(switchMap(() => attempt(left - 1)));
      }),
      catchError(err => {
        if (left <= 0) return throwError(() => err);
        return timer(delayMs).pipe(switchMap(() => attempt(left - 1)));
      })
    );
  return attempt(tries);
}

// === Methods ===
getSB(id: string) {
  return this.http.get<SBReport>(`${this.base}/reports/sb`, {
    params: { id, t: Date.now().toString() }
  });
}

getSBWithRetry(id: string, tries = 2, delayMs = 250): Observable<SBReport> {
  const attempt = (left: number): Observable<SBReport> =>
    this.getSB(id).pipe(
      // si por alguna razón viniera vacío, reintenta (no debería)
      switchMap(rep => {
        const ok = !!rep?.kind && rep.kind === 'sb';
        if (ok || left <= 0) return of(rep);
        return timer(delayMs).pipe(switchMap(() => attempt(left - 1)));
      }),
      catchError(err => {
        if (left <= 0) return throwError(() => err);
        return timer(delayMs).pipe(switchMap(() => attempt(left - 1)));
      })
    );
  return attempt(tries);
}


  getFileText(id: string, ruta: string): Observable<string> {
    const params = new HttpParams()
      .set('id', id)
      .set('ruta', ruta) // <-- aquí el cambio
      .set('t', Date.now().toString());

    return this.http.get(`${this.base}/reports/file`, {
      params,
      responseType: 'text'
    });
  }

  openFileInNewTab(id: string, ruta: string) {
  const url = `/api/reports/file?id=${encodeURIComponent(id)}&ruta=${encodeURIComponent(ruta)}`;


  const win = window.open(url, '_blank', 'noopener');
  if (!win) {
    const a = document.createElement('a');
    a.href = url;
    a.target = '_blank';
    a.rel = 'noopener';
    document.body.appendChild(a);
    a.click();
    a.remove();
  }
}


  getLS(id: string, ruta = '/') {
  return this.http.get<LSReport>(`${this.base}/reports/ls`, {
    params: { id, ruta, t: Date.now().toString() }
  });
}

getLSWithRetry(id: string, ruta = '/', tries = 2, delayMs = 250): Observable<LSReport> {
  const attempt = (left: number): Observable<LSReport> =>
    this.getLS(id, ruta).pipe(
      switchMap(rep => {
        const ok = Array.isArray(rep?.items); // si responde, listo
        if (ok || left <= 0) return of(rep);
        return timer(delayMs).pipe(switchMap(() => attempt(left - 1)));
      }),
      catchError(err => {
        if (left <= 0) return throwError(() => err);
        return timer(delayMs).pipe(switchMap(() => attempt(left - 1)));
      })
    );

  return attempt(tries);
}


getJournaling(id?: string) {
  const params: any = { t: Date.now().toString() };
  if (id) params.id = id;
  return this.http.get<JournalRow[]>('/api/reports/journaling', { params });
}

getJournalingWithRetry(id?: string, retries = 2, delayMs = 250) {
  return this.getJournaling(id).pipe(
    retryWhen(err$ => err$.pipe(delay(delayMs), take(retries)))
  );
}





}
