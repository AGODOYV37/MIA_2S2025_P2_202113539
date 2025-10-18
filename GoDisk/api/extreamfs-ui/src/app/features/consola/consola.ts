import { Component, ViewChild, ElementRef, NgZone, ChangeDetectorRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Commands } from '../../core/services/commands';
import { Reports, MBRReport, DiskReport, DiskSegment, InodeReport, InodeMini, BlockReport, BlockItem, TreeReport, TreeInode, SBReport, LSItem, LSReport } from '../../core/services/reports';
import { Observable, EMPTY, of, forkJoin } from 'rxjs';
import { finalize, map, switchMap } from 'rxjs/operators';
import { AuthService } from '../../core/services/auth';




type Block = { kind: DiskSegment['kind']; label: string; tip: string };
type RepKind = 'mbr' | 'disk' | 'block' | 'bm_inode' | 'bm_block' | 'tree' | 'ls'  ;



@Component({
  selector: 'app-consola',
  standalone: true,
  imports: [CommonModule, FormsModule],
  templateUrl: './consola.html',
  styleUrls: ['./consola.scss']
})
export class Consola {
  @ViewChild('smiaInput') smiaInput?: ElementRef<HTMLInputElement>;

  
  entrada = '';
  salida = '';
  cargando = false;
  


  // Modales (existentes)
  mbr: MBRReport | null = null;
  disk: DiskReport | null = null;
  inode: InodeReport | null = null;

  // Explorador de inodos (nuevo)
  inodesPartId = '';
  inodes: InodeMini[] = [];
  showInodes = false;
  inodeRoot: InodeReport | null = null; // tarjeta izquierda
  inodeSel: InodeReport | null = null;  // tarjeta derecha (seleccionada)
  inodesChain: InodeReport[] = [];
  maxChainNodes = 10; // opcional, límite por performance

  // Mosaicos (bloques)
  diskBlocks: Block[] = [];
  extBlocks: Block[] = [];
  inodeRuns: number[][] = [];


  blocks: BlockReport | null = null;
  blocksFlow: BlockItem[] = [];  
  showBlocks = false;

    // === TREE ===
  tree: TreeReport | null = null;
  showTree = false;
  treeSel = -1;  // inodo seleccionado

  sb: SBReport | null = null;
  dismissSB() { this.sb = null; }

  // estado nuevo
  ls: LSReport | null = null;


  get treeNode(): TreeInode | null {
  return this.tree?.nodes.find(n => n.index === this.treeSel) ?? null;
}
edgesFrom(idx: number) {
  return (this.tree?.edges || []).filter(e => e.parent === idx);
}
selectTreeInode(idx: number) {
  this.treeSel = idx;
}
dismissTREE() { this.showTree = false; this.tree = null; this.treeSel = -1; }


bmInodeText: string | null = null;
showBmInode = false;
dismissBmInode() { this.showBmInode = false; this.bmInodeText = null; }

bmBlockText: string | null = null;
showBmBlock = false;
dismissBmBlock() { this.showBmBlock = false; this.bmBlockText = null; }



dismissBLOCKS() { this.showBlocks = false; this.blocks = null; this.blocksFlow = []; }

dismissLS() { this.ls = null; }
  

  private actionSeq = 0;

    constructor(
    private api: Commands,
    private reports: Reports,
    public auth: AuthService,
    private zone: NgZone,
    private cdr: ChangeDetectorRef
  ) {}

  cargarArchivo(ev: Event) {
    const file = (ev.target as HTMLInputElement).files?.[0];
    if (!file) return;
    file.text().then(txt => (this.entrada = txt));
  }


  openSmia() {
  const el = this.smiaInput?.nativeElement;
  if (!el) return;
  el.value = '';          
  el.click();             
}



async onPickSmia(ev: Event) {
  const input = ev.target as HTMLInputElement;
  const files = input.files;
  if (!files || files.length === 0) return;

  try {
    const texts = await Promise.all(Array.from(files).map(f => f.text()));
    const merged = texts.join('\n').replace(/\r\n?/g, '\n');

    this.zone.run(() => {
      this.entrada = this.entrada ? (this.entrada + '\n\n' + merged) : merged;
      this.cdr.markForCheck(); 
    });
  } catch (e: any) {
    this.salida = `Error leyendo .smia: ${e?.message || e}`;
  } finally {
    input.value = '';
  }
}



 ejecutar() {

  
  if (!this.entrada.trim() || this.cargando) return;
  const script = this.entrada;
  const token = ++this.actionSeq;
  const fileQ = this.detectRepFile(script);


  // limpia TODO el estado (incluyendo blocks)
  this.mbr = null; this.disk = null; this.inode = null; this.sb = null; this.tree =null;
  this.inodesPartId = ''; this.inodes = []; this.showInodes = false;
  this.inodeRoot = null; this.inodeSel = null; this.inodesChain = [];
  this.diskBlocks = []; this.extBlocks = []; this.inodeRuns = [];
  this.blocks = null; this.blocksFlow = []; this.showBlocks = false;
  this.cargando = true; this.salida = '';
  this.bmInodeText = null; this.showBmInode = false;
  this.bmBlockText = null; this.showBmBlock = false;
  this.ls = null;


  // detectores
  const mbrId   = this.detectRep(script, 'mbr')?.id || null;
  const diskId  = this.detectRep(script, 'disk')?.id || null;
  const blockId = this.detectRep(script, 'block')?.id || null;
  const inodeQ  = this.detectRepInode(script);   // { id, ruta? }
  const inodesQ = this.detectRepInodes(script);  // { id }
  const bmInodeId = this.detectRep(script, 'bm_inode')?.id || null;
  const bmBlockId = this.detectRep(script, 'bm_block')?.id || null;
  const sbId   = this.detectRep(script, 'sb' as any /*extiende la union*/)?.id || null;
  const lsQ = this.detectRepLS(script);

    
  

  


  this.api.execute(script).pipe(
    switchMap(res => {
      if (token !== this.actionSeq) return EMPTY;
      this.salida = res?.output ?? '';

      // Pide lo necesario en paralelo
      const tasks: Observable<Record<string, any>>[] = [];
      if (mbrId)  tasks.push(this.reports.getMBRWithRetry(mbrId, 2, 250).pipe(map(m => ({ mbr: m }))));
      if (diskId) tasks.push(this.reports.getDiskWithRetry(diskId, 2, 250).pipe(map(d => ({ disk: d }))));
      if (blockId) tasks.push(this.reports.getBlocksWithRetry(blockId, 2, 250).pipe(map(b => ({ block: b })))); 
      
      this.tree = null; this.showTree = false; this.treeSel = -1;
      const treeId = this.detectRep(script, 'tree')?.id || null;


      if (treeId) tasks.push(this.reports.getTreeWithRetry(treeId, 2, 250).pipe(map(t => ({ tree: t }))));


      if (fileQ) {
        this.reports.openFileInNewTab(fileQ.id, fileQ.ruta);
      }


      if (inodeQ) {
        const call = inodeQ.ruta
          ? this.reports.getInodeWithRetry(inodeQ.id, inodeQ.ruta, 2, 250)
          : this.reports.getInodeWithRetry(inodeQ.id, '', 2, 250); // '' => backend usa default
        tasks.push(call.pipe(map(i => ({ inode: i }))));
      }

      if (bmInodeId) {
        tasks.push(this.reports.getBmInodeWithRetry(bmInodeId, 2, 250).pipe(map(txt => ({ bmInode: txt }))));
      }

      if (sbId)  tasks.push(this.reports.getSBWithRetry(sbId, 2, 250).pipe(map(s => ({ sb: s }))));


      if (inodesQ?.id) {
        tasks.push(this.reports.getInodes(inodesQ.id, 0).pipe(map(list => ({ inodesList: list }))));
      }

          if (bmBlockId) {
      tasks.push(this.reports.getBmBlockWithRetry(bmBlockId, 2, 250).pipe(map(txt => ({ bmBlock: txt }))));
    }

      if (lsQ?.id) {
        const r = lsQ.ruta ?? '/';
        tasks.push(this.reports.getLSWithRetry(lsQ.id, r, 2, 250).pipe(map(ls => ({ ls }))));
      }


      if (tasks.length === 0) return of(null);

      

      return forkJoin(tasks).pipe(
        map(parts => parts.reduce((acc, part) => ({ ...acc, ...part }), {} as Record<string, any>))
      );
    }),
    finalize(() => { if (token === this.actionSeq) this.cargando = false; })
  ).subscribe({
    next: (payload: any) => {
      if (token !== this.actionSeq) return;

      // MBR
      if (payload?.mbr?.partitions?.length) {
        this.mbr = { ...payload.mbr, partitions: [...payload.mbr.partitions] };
      }

      // DISK
      if (payload?.disk?.segments?.length) {
        const disk: DiskReport = payload.disk;
        this.disk = JSON.parse(JSON.stringify(disk));
        this.diskBlocks = this.makeBlocks(disk.segments, 100);
        if (disk.extended?.segments?.length) {
          this.extBlocks = this.makeBlocks(disk.extended.segments, 60);
        }
      }

      // INODE (modal simple)
      if (payload?.inode) {
        const inode: InodeReport = payload.inode;
        this.inode = JSON.parse(JSON.stringify(inode));
        this.inodeRuns = this.makeContiguousRuns(inode.blocks ?? []);
      }

      // INODES (explorador: cadena de tarjetas)
      if (payload?.inodesList?.items) {
        const list = payload.inodesList;
        this.inodesPartId = list.id || '';
        this.buildInodeChainFromList(list.items || []);
      }

      // BLOCKS (modal tipo tarjetas, SIEMPRE al nivel superior)
      if (payload?.block) {
        const rep = payload.block as BlockReport;
        const list = (rep as any).blocks ?? (rep as any).items ?? [];
        this.blocks = JSON.parse(JSON.stringify(rep));
        this.blocksFlow = Array.isArray(list) ? [...list].sort((a: any, b: any) => a.index - b.index) : [];
        this.showBlocks = true; // <-- abre el modal
      }

      if (typeof payload?.bmInode === 'string') {
              this.bmInodeText = payload.bmInode;
              this.showBmInode = true;
            }

      if (typeof payload?.bmBlock === 'string') {
            this.bmBlockText = payload.bmBlock;
            this.showBmBlock = true;
          }

      if (payload?.tree?.nodes?.length) {
        const rep: TreeReport = payload.tree;
        this.tree = JSON.parse(JSON.stringify(rep));
        this.treeSel = rep.root ?? rep.nodes[0]?.index ?? 0;
        this.showTree = true;
      }

            // SB
      if (payload?.sb) {
        this.sb = JSON.parse(JSON.stringify(payload.sb));
      }


      if (payload?.ls) {
        this.ls = payload.ls as LSReport;
      }







  

    },
    error: (err) => {
    if (token !== this.actionSeq) return;
    const msg =
      (err?.error && typeof err.error === 'object' && err.error.error) ||
      (typeof err?.error === 'string' ? err.error : '') ||
      err?.message ||
      'desconocido';
    this.salida = `Error: ${msg}`;
  }

  });
}




private detectRep(script: string, kind: RepKind): { id: string } | null {
  const lines = script.split('\n').map(s => s.trim()).filter(Boolean);
  for (const line of lines) {
    if (!/^rep\b/i.test(line)) continue;
    const name = /-name\s*=\s*"?([a-z_]+)"?/i.exec(line)?.[1]?.toLowerCase();
    if (name !== kind) continue;
    const id = /-id\s*=\s*"?([A-Za-z0-9]+)"?/i.exec(line)?.[1];
    if (id) return { id };
  }
  return null;
}


private pickFlag(line: string, key: string): string {
  const re = new RegExp(`-${key}\\s*=\\s*(?:"([^"]*)"|'([^']*)'|(\\S+))`, 'i');
  const m = re.exec(line);
  return (m?.[1] ?? m?.[2] ?? m?.[3] ?? '').trim();
}

private detectRepFile(script: string): { id: string; ruta: string } | null {
  const line = script.split('\n').map(s => s.trim()).find(s => /^rep\b/i.test(s));
  if (!line) return null;

  const name = this.pickFlag(line, 'name').toLowerCase();
  if (name !== 'file') return null;

  const id   = this.pickFlag(line, 'id');
  let ruta   = this.pickFlag(line, 'ruta')
            || this.pickFlag(line, 'path_file')
            || this.pickFlag(line, 'path_file_ls');

  if (!id || !ruta) return null;


  ruta = ruta.replace(/\\+/g, '/').replace(/\/{2,}/g, '/');
  return { id, ruta };
}



private detectRepLS(script: string): { id: string; ruta?: string } | null {
  const line = script.split('\n').map(s => s.trim()).find(s => /^rep\b/i.test(s));
  if (!line) return null;
  const name = /-name\s*=\s*"?([a-z_]+)"?/i.exec(line)?.[1]?.toLowerCase();
  if (name !== 'ls') return null;
  const id = /-id\s*=\s*"?([A-Za-z0-9]+)"?/i.exec(line)?.[1];
  const ruta = /-ruta\s*=\s*"?([^"]+)"?/i.exec(line)?.[1]; 
  return id ? (ruta ? { id, ruta } : { id }) : null;
}




  private buildInodeChainFromList(items: { index: number }[], max = this.maxChainNodes) {
  // abre el modal aunque aún no lleguen los nodos
  this.showInodes = true;
  this.inodesChain = [];

  if (!this.inodesPartId) return;

  const indices = (items || []).map(x => x.index).slice(0, max);
  if (!indices.length) return;

  // pide todos los inodos en paralelo manteniendo el orden
  forkJoin(indices.map(idx => this.reports.getInode(this.inodesPartId, String(idx))))
    .subscribe({
      next: nodes => {
        // clonado defensivo
        this.inodesChain = nodes.map(n => JSON.parse(JSON.stringify(n)));
      },
      error: err => {
        console.error('buildInodeChainFromList', err);
        this.inodesChain = [];
      }
    });
}


  // Detecta rep -name=inode -id=XXXX [-ruta=N]  (ruta opcional)
  private detectRepInode(script: string): { id: string; ruta?: string } | null {
    const line = script.split('\n').map(s => s.trim()).find(s => /^rep\b/i.test(s));
    if (!line) return null;
    const name = /-name\s*=\s*"?([a-z_]+)"?/i.exec(line)?.[1]?.toLowerCase();
    if (name !== 'inode') return null;
    const id = /-id\s*=\s*"?([A-Za-z0-9]+)"?/i.exec(line)?.[1];
    const ruta = /-ruta\s*=\s*"?([0-9/]+)"?/i.exec(line)?.[1]; // opcional
    return id ? (ruta ? { id, ruta } : { id }) : null;
  }

  // Detecta rep -name=inodes -id=XXXX  (explorador)
  private detectRepInodes(script: string): { id?: string } | null {
    const line = script.split('\n').map(s => s.trim()).find(s => /^rep\b/i.test(s));
    if (!line) return null;
    const name = /-name\s*=\s*"?([a-z_]+)"?/i.exec(line)?.[1]?.toLowerCase();
    if (name !== 'inodes') return null;
    const id = /-id\s*=\s*"?([A-Za-z0-9]+)"?/i.exec(line)?.[1];
    return id ? { id } : null;
  }

  get lsItemsSorted(): LSItem[] {
  const items = this.ls?.items ?? [];
  const order = (t: string) => (t === 'dir' ? 0 : t === 'file' ? 1 : 2);
  return [...items].sort((a, b) => {
    const byType = order(a.type) - order(b.type);
    return byType !== 0 ? byType : a.name.localeCompare(b.name);
  });
}


  // Carga detalle de un inodo y actualiza tarjetas + corridas
  selectInode(idx: number) {
    if (!this.inodesPartId) return;
    this.reports.getInode(this.inodesPartId, String(idx)).subscribe({
      next: rep => {
        // La primera vez, fija también la tarjeta izquierda (root)
        if (!this.inodeRoot) this.inodeRoot = rep;
        this.inodeSel = rep;
        this.inodeRuns = this.makeContiguousRuns(rep.blocks ?? []);
      },
      error: err => console.error(err)
    });
  }

  closeInodes() {
    this.showInodes = false;
    this.inodes = [];
    this.inodeRoot = null;
    this.inodeSel = null;
    this.inodeRuns = [];
    this.inodesChain = [];
  }

  // Convierte segmentos en bloques discretos para mosaico
  private makeBlocks(segments: DiskSegment[], total = 100): Block[] {
    const segs = segments.filter(s => s.percent > 0);
    const perc = segs.map(s => s.percent);
    const sum = perc.reduce((a,b)=>a+b,0) || 1;

    const rawCounts = segs.map(s => Math.max(0, Math.round((s.percent/sum)*total)));
    let assigned = rawCounts.reduce((a,b)=>a+b,0);

    const byError = segs
      .map((s,i)=>({i, err: ((s.percent/sum)*total) - rawCounts[i]}))
      .sort((a,b)=> b.err - a.err);

    while (assigned < total && byError.length) { rawCounts[byError[0].i]++; assigned++; }
    while (assigned > total && byError.length) { rawCounts[byError[byError.length-1].i]--; assigned--; }

    const out: Block[] = [];
    segs.forEach((s, idx) => {
      const n = rawCounts[idx];
      const tip = `${s.label || s.kind} • ${s.kind} • ${s.percent.toFixed(2)}%  [${s.start}..${s.end}]`;
      for (let k=0;k<n;k++) out.push({ kind: s.kind, label: s.label || s.kind, tip });
    });
    if (out.length < total) out.push(...Array.from({length: total - out.length},()=>({kind:'FREE' as const, label:'libre', tip:'libre'})));
    if (out.length > total) out.length = total;
    return out;
  }


  private makeContiguousRuns(blocks: number[]): number[][] {
    const arr = Array.from(new Set((blocks || []).filter(n => n > 0))).sort((a,b)=>a-b);
    const runs: number[][] = [];
    let cur: number[] = [];
    for (let i = 0; i < arr.length; i++) {
      if (i === 0 || arr[i] === arr[i-1] + 1) cur.push(arr[i]);
      else { if (cur.length) runs.push(cur); cur = [arr[i]]; }
    }
    if (cur.length) runs.push(cur);
    return runs;
  }


get salidaConPrefijo(): string {

  return (this.salida ?? '').replace(/^/gm, '> ');

}


  dismissMBR()  { this.mbr = null; }
  dismissDISK() { this.disk = null;  this.diskBlocks=[]; this.extBlocks=[]; }

  trackRow = (_: number, p: any) => `${p.index}-${p.start}-${p.size}`;
  trackLS(_: number, it: LSItem): string {
  return `${it.inode}-${it.name}`;
}
}
