import { Component, OnInit, NgZone, ChangeDetectorRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { MountsService, MountView } from '../../core/services/mount';
import { finalize } from 'rxjs/operators';

@Component({
  selector: 'app-visualizador',
  standalone: true,
  imports: [CommonModule, RouterLink],
  templateUrl: './discos.html',
  styleUrls: ['./discos.scss']
})
export class VisualizadorComponent implements OnInit {
  loading = true;
  error = '';
  mounts: MountView[] = [];

  constructor(
    private mountsSvc: MountsService,
    private zone: NgZone,
    private cdr: ChangeDetectorRef
  ) {}

  ngOnInit(): void {
    this.loading = true;
    this.mountsSvc.getAll()
      .pipe(finalize(() => {
        // apaga el spinner pase lo que pase
        this.zone.run(() => { this.loading = false; this.cdr.detectChanges(); });
      }))
      .subscribe({
        next: (list) => {

          this.zone.run(() => {
            const norm = (p: string) => (p || '').replace(/\\+/g, '/').toLowerCase().trim();


            const byDisk = new Map<string, MountView>();
            for (const m of (list ?? [])) {
              const key = norm(m.diskPath);
              if (!byDisk.has(key)) byDisk.set(key, m);
            }

            const onlyDisks = Array.from(byDisk.values());
            this.mounts = onlyDisks.sort((a, b) =>
              this.baseName(a.diskPath).localeCompare(this.baseName(b.diskPath))
            );

            this.error = '';
            this.cdr.detectChanges(); 
          });
        },
        error: (err) => {
          this.zone.run(() => {
            this.error = (err?.error?.error || err?.message || 'Error cargando montajes');
            this.mounts = [];
            this.cdr.detectChanges();
          });
        }
      });
  }

  baseName(p: string): string {
    if (!p) return '';
    const clean = p.replace(/\\+/g, '/');
    const parts = clean.split('/');
    return parts[parts.length - 1] || p;
  }
}
