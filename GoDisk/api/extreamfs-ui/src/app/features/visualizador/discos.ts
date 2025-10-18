import { Component, OnInit, NgZone, ChangeDetectorRef } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { MountsService, MountView } from '../../core/services/mount';

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
    this.cdr.detectChanges(); // asegura que el "Actualizando…" se pinte

    this.mountsSvc.getAll().subscribe({
      next: (list) => {
        // Fuerza a correr dentro de Angular y refrescar la vista
        this.zone.run(() => {
          this.mounts = list ?? [];
          this.loading = false;
          this.cdr.detectChanges();
        });
      },
      error: (err) => {
        this.zone.run(() => {
          this.error = (err?.error?.error || err?.message || 'Error cargando montajes');
          this.mounts = [];
          this.loading = false;
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
