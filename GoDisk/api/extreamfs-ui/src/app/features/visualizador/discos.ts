import { Component, OnInit } from '@angular/core';
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

  constructor(private mountsSvc: MountsService) {}

  ngOnInit(): void {
    this.mountsSvc.getAll().subscribe({
      next: (list) => {
        this.mounts = list ?? [];
        this.loading = false;
      },
      error: (err) => {
        this.error = (err?.error?.error || err?.message || 'Error cargando montajes');
        this.mounts = [];
        this.loading = false;        
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
